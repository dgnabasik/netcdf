echo dgnabasik
git pull https://github.com/dgnabasik/netcdf
echo -n "push?"
read
git add --all :/
git commit -am "Release 1.0.1"
git push -u origin main
