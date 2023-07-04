read -p 'Script parameters: fileName key'
curl -o /tmp/$1 http://localhost:8080/get/$2
ls -l /tmp/$1
md5sum /tmp/$1

