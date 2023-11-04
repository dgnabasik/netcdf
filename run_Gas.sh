go build
read -p "the program built."
./netcdf /home/david/Documents/digital-twins/AMP/NaturalGas_FRG.csv unix_ts create insert
read -p "???"
./netcdf /home/david/Documents/digital-twins/AMP/NaturalGas_WHG.csv unix_ts create insert


