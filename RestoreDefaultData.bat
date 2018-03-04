@echo off
SETLOCAL
cd .\app\assets\json\default
copy largeDevDirSet.json ..\data.json > temp.txt
del temp.txt
echo DEMO DATASET RESTORED
timeout 1