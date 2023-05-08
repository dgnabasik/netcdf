echo dgnabasik in netcdf
eval "$(ssh-agent -s)"
git remote -v
ssh -T -ai /home/david/.ssh/netcdf_ed25519 git@github.com
git pull git@github.com:dgnabasik/netcdf.git
echo -n "push?"
read
git add --all :/
git commit -am "Release 1.0.1"
git push -u origin main
