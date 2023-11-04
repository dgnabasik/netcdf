go build
read -p "the program built."
./netcdf /home/david/Documents/digital-twins/AMP/Water_WHW.csv unix_ts create insert
read -p "???"
./netcdf /home/david/Documents/digital-twins/AMP/Water_HTW.csv unix_ts create insert
./netcdf /home/david/Documents/digital-twins/AMP/Water_DWW.csv unix_ts create insert

