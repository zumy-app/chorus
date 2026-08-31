$errors = $null
[System.Management.Automation.Language.Parser]::ParseFile('C:\dev\chorus\start-android.ps1', [ref] $null, [ref] $errors)
if ($errors.Count) { 'SYNTAX ERRORS' ; $errors | ForEach-Object { Write-Error $_.Message } }
else { 'SYNTAX OK' }