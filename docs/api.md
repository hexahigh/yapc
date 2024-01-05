# Api
## /store
### POST
Body must be multipart/form-data and have a field named file containing the file.<br>
A 201 response means the file was succesfully uploaded and saved.
A 200 response means the file was succesfully uploaded but not saved because it already exists.
#### Curl example:
```
curl -X POST -F file=@/path/to/file http://localhost:8080/store
```

## /get
### GET
Returns the file with the given hash.
#### Curl example:
```
curl http://localhost:8080/get/00000000000
```

## /stats
### GET
Returns statistics about the server.
#### Curl example:
```
curl http://localhost:8080/stats
```