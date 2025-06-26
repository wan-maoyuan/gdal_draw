package gdaldraw

import (
	"bufio"
	"fmt"
	"image"
	"math"
	"os"

	"image/color"
	"image/png"
)

const (
	MinValue        = -20037508.34
	MaxValue        = 20037508.34
	PictureAccuracy = 2049 // 图片精度
	StepValue       = (MaxValue - MinValue) / PictureAccuracy
)

type ColorFunc func(img *image.RGBA, x, y int, value float64)

type DounbleColorFunc func(img *image.RGBA, x, y int, value1, value2 float64)

// 默认着色函数填充红色，这个函数可以自定义
func DefaultColorFunc(img *image.RGBA, x, y int, value float64) {
	var colorValue = uint8(value)

	img.SetRGBA(x, y, color.RGBA{
		colorValue, 0, 0, 255,
	})
}

func Draw3857(data *Data, colorFunc ColorFunc) error {
	if err := data.check(); err != nil {
		return err
	}

	img := image.NewRGBA(image.Rect(0, 0, 2049, 2049))
	for xIndex := 0; xIndex <= 2048; xIndex++ {
		for yIndex := 0; yIndex <= 2048; yIndex++ {
			x := MinValue + StepValue*float64(xIndex)
			y := MinValue + StepValue*float64(yIndex)

			lat, lon := convert3857To4326(x, y)
			latIndex := int((lat + 90) / data.Accuracy)
			lonIndex := int((lon + 180) / data.Accuracy)

			value := data.ValueList[latIndex][lonIndex]
			colorFunc(img, xIndex, yIndex, value)
		}
	}

	file, err := os.OpenFile(data.OutFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("create out file: %s error: %v", data.OutFilePath, err)
	}
	defer file.Close()

	fileWriter := bufio.NewWriter(file)
	if err := png.Encode(fileWriter, img); err != nil {
		return fmt.Errorf("png encode data to file error: %v", err)
	}

	if err := fileWriter.Flush(); err != nil {
		return fmt.Errorf("flush data to out file error: %v", err)
	}

	return nil
}

func DrawDouble3857(data *DoubleData, colorFunc DounbleColorFunc) error {
	if err := data.check(); err != nil {
		return err
	}

	img := image.NewRGBA(image.Rect(0, 0, 2049, 2049))
	for xIndex := 0; xIndex <= 2048; xIndex++ {
		for yIndex := 0; yIndex <= 2048; yIndex++ {
			x := MinValue + StepValue*float64(xIndex)
			y := MinValue + StepValue*float64(yIndex)

			lat, lon := convert3857To4326(x, y)
			latIndex := int((lat + 90) / data.Accuracy)
			lonIndex := int((lon + 180) / data.Accuracy)

			value1 := data.Value1List[latIndex][lonIndex]
			value2 := data.Value2List[latIndex][lonIndex]

			colorFunc(img, xIndex, yIndex, value1, value2)
		}
	}

	file, err := os.OpenFile(data.OutFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("create out file: %s error: %v", data.OutFilePath, err)
	}
	defer file.Close()

	fileWriter := bufio.NewWriter(file)
	if err := png.Encode(fileWriter, img); err != nil {
		return fmt.Errorf("png encode data to file error: %v", err)
	}

	if err := fileWriter.Flush(); err != nil {
		return fmt.Errorf("flush data to out file error: %v", err)
	}

	return nil
}

func convert3857To4326(x, y float64) (lat, lon float64) {
	lon = x * 180.0 / MaxValue
	lat = math.Atan(math.Exp(y*math.Pi/MaxValue))*360.0/math.Pi - 90.0

	return
}

func convert4326To3857(lat, lon float64) (x, y float64) {
	x = lon * MaxValue / 180
	y = math.Log(math.Tan((90+lat)*math.Pi/360)) / (math.Pi / 180)
	y = y * MaxValue / 180

	return
}

func DrawIrregular3857(data *IrregularData, colorFunc ColorFunc) error {
	if err := data.check(); err != nil {
		return err
	}

	img := image.NewRGBA(image.Rect(0, 0, 2049, 2049))
	for index := 0; index < len(data.LatList); index++ {
		lat := data.LatList[index]
		lon := data.LonList[index]
		x, y := convert4326To3857(lat, lon)

		xIndex := int((x - MinValue) / StepValue)
		yIndex := 2049 - int((y-MinValue)/StepValue)

		value := data.ValueList[index]
		colorFunc(img, xIndex, yIndex, value)
	}

	file, err := os.OpenFile(data.OutFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("create out file: %s error: %v", data.OutFilePath, err)
	}
	defer file.Close()

	fileWriter := bufio.NewWriter(file)
	if err := png.Encode(fileWriter, img); err != nil {
		return fmt.Errorf("png encode data to file error: %v", err)
	}

	if err := fileWriter.Flush(); err != nil {
		return fmt.Errorf("flush data to out file error: %v", err)
	}

	return nil
}
