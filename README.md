README.md for netcdf repository:: 2023-07-03
!!!!!!!!!!!! The two versions of filesystem.go have to be synchronized: 
  /home/david/Documents/digital-twins/micro.services/filesystem/filesystem.go
  /home/david/Documents/digital-twins/netcdf/filesystem/filesystem.go
Program goals:
1) Provide a customizable streaming sensor data publish/subscribe server for comsumption by Python, Golang, Java clients.
2) Automatically generate SAREF-derived OWL class per data stream source; semantically enrich SAREF data.
3) Provide test data for digital-twin simulations. 

Configuration:
IotDB v1.10
Golang v1.20.5
GraphDB v10.2.1-1
RDF4J v3.73
Ubuntu 22.04.2 LTS with Linux kernel 5.19.17-051917-generic.
Intel NUC 10i5FNH: 1Tb disk, 32 Gb RAM.
Java v11.0.2

Available Datasets
1) Ecobee Thermostat dataset: ../digital-twins/ecobee
https://bbd.labworks.org/ds/bbd/ecobee https://www.osti.gov/dataexplorer/biblio/dataset/1854924 
Data records include: thermostat setpoints and motions, the actual indoor temperature, relative humidity, run-time for HVAC equipment and devices; home characteristics (e.g., city, province, floor area, number of floors, vintage, etc.), system characteristics (e.g., heating and cooling types and installation status), occupant characteristics (e.g., number of occupants); outside temperature and humidity data.

2) ToN_IoT datasets (telemetry datasets of IoT and IIoT sensors): ../digital-twins/ton.iot
https://cloudstor.aarnet.edu.au/plus/s/ds5zW91vdgjEj9i 
Data records include: Modbus, motion & light, Thermostat, weather data. Modbus is a data communications protocol originally published by Modicon in 1979 for use with its programmable logic controllers (PLCs). Modbus has become a de facto standard communication protocol and is now a commonly available means of connecting industrial electronic devices.[1] https://en.wikipedia.org/wiki/Modbus 

3) Smart Home Dataset with weather Information: ../digital-twins/smart.home.weather/homeC.csv
https://www.kaggle.com/datasets/taranvee/smart-home-dataset-with-weather-information
Home energy use measurements and weather data.

4) Open Powercom System Dataset: ../digital-twins/opsd.household
https://open-power-system-data.org/ 
Household load and solar generation in minutely to hourly resolution.

5) Commercial Building Energy Dataset (IITD): ../digital-twins/combed.iiitd
http://combed.github.io/
Commercial Building current, energy, power, lifts, lights. 

6) Almanac of Minutely Power datasets: ../digital-twins/AMP 
http://ampds.org/
Electricity, natural gas, water quality, climate.


