package gdaldraw

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fogleman/contourmap"
)

type ContourPoint struct {
	X, Y float64
}

type ContourLine struct {
	Value  float64
	Points []ContourPoint
}

// ====== 1. 边界扩展 ======
func expandGridForLongitude(data *ContourLinesData) ([][]float64, []float64) {
	rows := len(data.ValueList)
	cols := len(data.ValueList[0])
	// 新经度列表
	newLonList := make([]float64, cols+2)
	newLonList[0] = data.LonList[0] - 10 // -190
	copy(newLonList[1:], data.LonList)
	newLonList[len(newLonList)-1] = data.LonList[len(data.LonList)-1] + 10 // 190
	// 新值列表
	newValueList := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		newValueList[i] = make([]float64, cols+2)
		newValueList[i][0] = data.ValueList[i][0] // 左边界直接复制
		copy(newValueList[i][1:], data.ValueList[i])
		newValueList[i][cols+1] = data.ValueList[i][cols-1] // 右边界直接复制
	}
	return newValueList, newLonList
}

// ====== 2. 等值线生成（使用 contourmap） ======
func generateContourLinesWithContourMap(data *ContourLinesData) ([]ContourLine, error) {
	grid, lonList := expandGridForLongitude(data)
	latList := data.LatList
	rows := len(grid)
	cols := len(grid[0])
	flat := make([]float64, 0, rows*cols)
	for i := 0; i < rows; i++ {
		flat = append(flat, grid[i]...)
	}
	cm := contourmap.FromFloat64s(cols, rows, flat)

	// 计算等值线值
	min, max := grid[0][0], grid[0][0]
	for i := range grid {
		for j := range grid[i] {
			if grid[i][j] < min {
				min = grid[i][j]
			}
			if grid[i][j] > max {
				max = grid[i][j]
			}
		}
	}
	var values []float64
	for v := min; v <= max; v += data.Step {
		values = append(values, v)
	}

	var result []ContourLine
	for _, v := range values {
		paths := cm.Contours(v)
		for _, path := range paths {
			if len(path) < 2 {
				continue
			}
			line := make([]ContourPoint, len(path))
			for i, pt := range path {
				lon := interpolate(lonList, pt.X)
				lat := interpolate(latList, pt.Y)
				line[i] = ContourPoint{X: lon, Y: lat}
			}
			smooth := smoothContourLine(line, 5)
			result = append(result, ContourLine{
				Value:  v,
				Points: smooth,
			})
		}
	}
	return result, nil
}

// ====== 3. 等值线平滑 ======
func smoothContourLine(line []ContourPoint, window int) []ContourPoint {
	if len(line) <= window {
		return line
	}
	var result []ContourPoint
	w := window
	for i := 0; i < len(line); i++ {
		sumX, sumY, count := 0.0, 0.0, 0
		for j := i - w/2; j <= i+w/2; j++ {
			if j >= 0 && j < len(line) {
				sumX += line[j].X
				sumY += line[j].Y
				count++
			}
		}
		result = append(result, ContourPoint{X: sumX / float64(count), Y: sumY / float64(count)})
	}
	return result
}

// ====== 4. GeoJSON 输出 ======
type geojsonFeature struct {
	Type       string          `json:"type"`
	Geometry   geojsonGeometry `json:"geometry"`
	Properties map[string]any  `json:"properties"`
}
type geojsonGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type geojson struct {
	Type     string           `json:"type"`
	Features []geojsonFeature `json:"features"`
}

func saveAsGeoJSON(contourLines []ContourLine, filePath string) error {
	gj := geojson{
		Type:     "FeatureCollection",
		Features: []geojsonFeature{},
	}
	for _, line := range contourLines {
		if len(line.Points) < 2 {
			continue
		}
		coords := make([][]float64, len(line.Points))
		for i, pt := range line.Points {
			coords[i] = []float64{pt.X, pt.Y}
		}
		gj.Features = append(gj.Features, geojsonFeature{
			Type: "Feature",
			Geometry: geojsonGeometry{
				Type:        "LineString",
				Coordinates: coords,
			},
			Properties: map[string]any{"value": line.Value},
		})
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(gj)
}

// ====== 5. 辅助函数 ======
// 插值（用于经纬度）
func interpolate(list []float64, idx float64) float64 {
	i := int(idx)
	if i < 0 {
		i = 0
	}
	if i >= len(list)-1 {
		i = len(list) - 2
	}
	frac := idx - float64(i)
	return list[i]*(1-frac) + list[i+1]*frac
}

// ====== 6. 主入口 ======
func DrawContourLines(data *ContourLinesData) error {
	contourLines, err := generateContourLinesWithContourMap(data)
	if err != nil {
		return fmt.Errorf("generate contour lines error: %v", err)
	}
	if err := saveAsGeoJSON(contourLines, data.OutFilePath); err != nil {
		return fmt.Errorf("save geojson error: %v", err)
	}
	return nil
}
