#!/bin/sh

wslpath="/mnt/c/Pictures/xxx.jpg"
winpath="$(wslpath -w $wslpath)"

powershell.exe -NoLogo -NoProfile -NonInteractive -Command "
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

\$img = [System.Drawing.Image]::FromFile('$winpath')

\$data = New-Object System.Windows.Forms.DataObject

# CF_BITMAP (Image)
\$data.SetImage(\$img)

# CF_UNICODETEXT (Text)
\$data.SetText('$wslpath', [System.Windows.Forms.TextDataFormat]::UnicodeText)

# CF_HDROP (FileDropList)
\$files = New-Object System.Collections.Specialized.StringCollection
\$files.Add('$winpath')
\$data.SetFileDropList(\$files)

[System.Windows.Forms.Clipboard]::SetDataObject(\$data, \$true)
"
