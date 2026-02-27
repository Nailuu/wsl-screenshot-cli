#!/bin/sh
set -eu

IMAGE_FILE_NAME="Screenshot $(date '+%Y-%m-%d %H%M%S').png"
WSL_BASE_DIR="/tmp/wsl-screenshot-cli"
WSL_IMAGE_TMP_PATH="$WSL_BASE_DIR/$IMAGE_FILE_NAME"

mkdir -p "$WSL_BASE_DIR"

WIN_IMAGE_TMP_PATH="$(wslpath -w "$WSL_IMAGE_TMP_PATH")"

PS_CMD="
Add-Type -AssemblyName System.Windows.Forms,System.Drawing;

\$image = [System.Windows.Forms.Clipboard]::GetImage();
if (-not \$image) { throw 'No image found in clipboard.' }

try {
    \$image.Save('$WIN_IMAGE_TMP_PATH', [System.Drawing.Imaging.ImageFormat]::Png);

    \$data = New-Object System.Windows.Forms.DataObject;
    \$data.SetImage(\$image);
    \$data.SetText('$WSL_IMAGE_TMP_PATH', [System.Windows.Forms.TextDataFormat]::UnicodeText);

    \$files = New-Object System.Collections.Specialized.StringCollection;
    [void]\$files.Add('$WIN_IMAGE_TMP_PATH');
    \$data.SetFileDropList(\$files);

    [System.Windows.Forms.Clipboard]::SetDataObject(\$data, \$true);

    Write-Output 'Saved.';
}
finally {
    \$image.Dispose();
}
"

powershell.exe -STA -NoLogo -NoProfile -NonInteractive -Command "$PS_CMD"
