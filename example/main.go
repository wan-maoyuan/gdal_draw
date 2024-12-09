package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/batchatco/go-native-netcdf/netcdf"
	"github.com/wan-maoyuan/gdaldraw"
)

const NcPath = "./nc_files/t2m_2022_01_01_ERA5.nc"

func main() {
	data, err := ReadDataFromNcFile(NcPath)
	if err != nil {
		log.Fatalf("read nc file data: %v", err)
	}

	data.Accuracy = 0.25
	data.OutFilePath = "./temp_3857.png"

	minTemp := float64(-50)
	maxTemp := float64(50)

	gdaldraw.Draw3857(data, func(img *image.RGBA, x, y int, value float64) {
		var colorValue = uint8((value - minTemp) / (maxTemp - minTemp) * 255)

		img.SetRGBA(x, y, color.RGBA{
			colorValue, 0, 0, colorValue,
		})
	})
}

func ReadDataFromNcFile(path string) (*gdaldraw.Data, error) {
	group, err := netcdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read nc file: %v", err)
	}
	defer group.Close()

	latVariable, err := group.GetVariable("latitude")
	if err != nil {
		return nil, fmt.Errorf("get variable latitude: %v", err)
	}
	latList := latVariable.Values.([]float32)

	lonVariable, err := group.GetVariable("longitude")
	if err != nil {
		return nil, fmt.Errorf("get variable longitude: %v", err)
	}
	lonList := lonVariable.Values.([]float32)

	tempVariable, err := group.GetVariable("t2m")
	if err != nil {
		return nil, fmt.Errorf("get variable t2m: %v", err)
	}
	tempList := tempVariable.Values.([][][]float32)[0]

	var newLat []float64
	for _, lat := range latList {
		newLat = append(newLat, float64(lat))
	}

	var newLon []float64
	for _, lon := range lonList {
		newLon = append(newLon, float64(lon))
	}

	var valueList [][]float64
	for _, temps := range tempList {
		var valueItem []float64

		for _, item := range temps {
			valueItem = append(valueItem, float64(item))
		}

		valueList = append(valueList, valueItem)
	}

	return &gdaldraw.Data{
		LatList:   newLat,
		LonList:   newLon,
		ValueList: valueList,
	}, nil
}
