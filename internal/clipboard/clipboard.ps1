Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# Uses pre-compiled .NET Clipboard APIs for change detection instead of a
# runtime-compiled C# class (Add-Type -TypeDefinition). Runtime C# compilation
# requires csc.exe, which EDR products (SentinelOne, CrowdStrike, etc.) block
# as a "living off the land" binary. The pre-compiled APIs work everywhere.
#
# The main loop still pumps messages via DoEvents() so the STA thread stays
# responsive, preventing Explorer/Snipping Tool freezes during OLE/COM
# clipboard operations.

[Console]::Out.WriteLine("READY")
[Console]::Out.Flush()

$readTask = [Console]::In.ReadLineAsync()

while ($true) {
    # Pump Windows messages so other apps' clipboard operations (OLE/COM)
    # don't time out waiting for our STA apartment to respond.
    [System.Windows.Forms.Application]::DoEvents()

    if (-not $readTask.IsCompleted) {
        Start-Sleep -Milliseconds 10
        continue
    }

    $line = $readTask.Result
    if ($line -eq $null -or $line -eq "EXIT") { break }

    if ($line -eq "CHECK") {
        try {
            # Skip if no image on clipboard
            if (-not [System.Windows.Forms.Clipboard]::ContainsImage()) {
                [Console]::Out.WriteLine("NONE")
                [Console]::Out.Flush()
                $readTask = [Console]::In.ReadLineAsync()
                continue
            }

            # Skip clipboard from spreadsheet apps (Excel, Google Sheets, etc.)
            # These apps copy cells as images but also include data formats like
            # CSV, HTML, or XML Spreadsheet that pure screenshots never have.
            $dataObj = [System.Windows.Forms.Clipboard]::GetDataObject()
            if ($dataObj -ne $null) {
                $formats = $dataObj.GetFormats()
                if ($formats -contains "XML Spreadsheet" -or
                    $formats -contains "Csv" -or
                    ($formats -contains "HTML Format" -and [System.Windows.Forms.Clipboard]::ContainsText())) {
                    [Console]::Out.WriteLine("NONE")
                    [Console]::Out.Flush()
                    $readTask = [Console]::In.ReadLineAsync()
                    continue
                }
            }

            # Fingerprint: if clipboard has image + text + file drop, it still
            # holds our previous enriched write (SetImage + SetText + SetFileDropList).
            # Snipping Tool / Win+Shift+S only sets the image format, so the
            # presence of all three means no new capture has arrived.
            if ([System.Windows.Forms.Clipboard]::ContainsText() -and
                [System.Windows.Forms.Clipboard]::ContainsFileDropList()) {
                [Console]::Out.WriteLine("NONE")
                [Console]::Out.Flush()
                $readTask = [Console]::In.ReadLineAsync()
                continue
            }

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

    $readTask = [Console]::In.ReadLineAsync()
}
