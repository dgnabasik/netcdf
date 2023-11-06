./netcdf /home/david/Documents/digital-twins/ton.iot/processed/IoT_Modbus.csv utc_timestamp insert
read -p "?"
./netcdf /home/david/Documents/digital-twins/ton.iot/processed/IoT_Weather.csv utc_timestamp insert
./netcdf /home/david/Documents/digital-twins/ton.iot/processed/IoT_Motion_Light.csv utc_timestamp insert
./netcdf /home/david/Documents/digital-twins/ton.iot/processed/IoT_Thermostat.csv utc_timestamp insert
echo Done!

#INSERT INTO root.toniot.synthetic.IoT_Modbus (time, Utc_timestamp,Datetime,FC1_Read_Input_Register,FC2_Read_Discrete_Value,FC3_Read_Holding_Register,FC4_Read_Coil,Label,Type,DatasetName ) ALIGNED VALUES (1556086094,1556086094,2019-04-24T06:08:14Z,51475,42123,20087,53134,0,'normal','IoT_Modbus'),(1556086094,1556086094,2019-04-24T06:08:14Z,51478,35547,40595,33768,0,'normal','IoT_Modbus'),(1556086094,1556086094,2019-04-24T06:08:14Z,51479,15581,45740,46059,0,'normal','IoT_Modbus'),(1556086094,1556086094,2019-04-24T06:08:14Z,51483,45969,44094,4740,0,'normal','IoT_Modbus');
