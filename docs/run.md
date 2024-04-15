# How to use the run:upload flag
The run:upload flag allows you to run external commands once a file has been uploaded, this can be used to scan a file for viruses, move it to an external server for backup, or perform other actions.

It is quite simple, lets say you wanted to scan a file using Clamav and delete it if it is infected. Then you can add the following flag to your run command: ```-run:upload "clamscan -d {%FULLPATH%}"```

The program will then transform {%FULLPATH%} into the path to the uploaded file and run the command.

You can use the same flag multiple times to run multiple commands on the same file.
```-run:upload "mv {%FULLPATH%} /tmp && clamscan -d {%FULLPATH%}"```

The full list of arguments is:

| Flag | Description |
| --- | --- |
| {%FILEPATH%} | The path to the uploaded file. |
| {%FULLPATH%} | The full path to the uploaded file. |
| {%SHA256%} | The SHA256 hash of the uploaded file. |
| {%SHA1%} | The SHA1 hash of the uploaded file. |
| {%MD5%} | The MD5 hash of the uploaded file. |
| {%CRC32%} | The CRC32 hash of the uploaded file. |
| {%AHASH%} | The AHash hash of the uploaded file. |
| {%DHASH%} | The DHash hash of the uploaded file. |
| {%CONTENTTYPE%} | The content type of the uploaded file. |