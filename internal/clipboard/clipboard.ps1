Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
# GetClipboardSequenceNumber returns a counter that increments on every clipboard
# change. Unlike GetImage(), it does NOT open or lock the clipboard, so polling it
# at high frequency has zero contention with other apps writing to the clipboard.
Add-Type -MemberDefinition '[DllImport("user32.dll")] public static extern uint GetClipboardSequenceNumber();' -Name NativeMethods -Namespace Win32

$lastSeq = [uint32]0

[Console]::Out.WriteLine("READY")
[Console]::Out.Flush()

while ($true) {
    $line = [Console]::ReadLine()
    if ($line -eq $null -or $line -eq "EXIT") { break }

    if ($line -eq "CHECK") {
        try {
            # Fast path: skip clipboard access entirely if nothing changed since last check
            $seq = [Win32.NativeMethods]::GetClipboardSequenceNumber()
            if ($seq -eq $lastSeq) {
                [Console]::Out.WriteLine("NONE")
                [Console]::Out.Flush()
                continue
            }
            $lastSeq = $seq

            $img = [System.Windows.Forms.Clipboard]::GetImage()
            if ($img -eq $null) {
                [Console]::Out.WriteLine("NONE")
                [Console]::Out.Flush()
            } else {
                try {
                    $ms = New-Object System.IO.MemoryStream
                    $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
                    $bytes = $ms.ToArray()
                    $ms.Dispose()
                    $b64 = [Convert]::ToBase64String($bytes)
                    [Console]::Out.WriteLine("IMAGE")
                    [Console]::Out.WriteLine($b64)
                    [Console]::Out.WriteLine("END")
                    [Console]::Out.Flush()
                } finally {
                    $img.Dispose()
                }
            }
        } catch {
            [Console]::Out.WriteLine("NONE")
            [Console]::Out.Flush()
        }
    }
    elseif ($line.StartsWith("UPDATE|")) {
        $parts = $line.Split("|")
        $wslPath = $parts[1]
        $winPath = $parts[2]
        try {
            $img = [System.Drawing.Image]::FromFile($winPath)
            try {
                $data = New-Object System.Windows.Forms.DataObject
                $data.SetImage($img)
                $data.SetText($wslPath, [System.Windows.Forms.TextDataFormat]::UnicodeText)

                $files = New-Object System.Collections.Specialized.StringCollection
                [void]$files.Add($winPath)
                $data.SetFileDropList($files)

                [System.Windows.Forms.Clipboard]::SetDataObject($data, $true)
                # Capture new seq so the next CHECK doesn't re-read our own write
                $lastSeq = [Win32.NativeMethods]::GetClipboardSequenceNumber()
                [Console]::Out.WriteLine("OK")
                [Console]::Out.Flush()
            } finally {
                $img.Dispose()
            }
        } catch {
            [Console]::Out.WriteLine("ERR|" + $_.Exception.Message)
            [Console]::Out.Flush()
        }
    }
}
