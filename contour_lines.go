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

	// 计算经度间隔
	lonStep := (data.LonList[len(data.LonList)-1] - data.LonList[0]) / float64(cols-1)

	// 新经度列表
	newLonList := make([]float64, cols+2)
	newLonList[0] = data.LonList[0] - lonStep
	copy(newLonList[1:], data.LonList)
	newLonList[len(newLonList)-1] = data.LonList[len(data.LonList)-1] + lonStep

	// 新值列表 - 关键修改：利用经度的周期性
	// 左边界应该使用右边的数据，右边界应该使用左边的数据
	newValueList := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		newValueList[i] = make([]float64, cols+2)
		newValueList[i][0] = data.ValueList[i][cols-1] // 左边界使用最右侧的值（周期性）
		copy(newValueList[i][1:], data.ValueList[i])
		newValueList[i][cols+1] = data.ValueList[i][0] // 右边界使用最左侧的值（周期性）
	}
	return newValueList, newLonList
}

// ====== 2. 等值线生成（使用 contourmap） ======
func generateContourLinesWithContourMap(data *ContourLinesData) ([]ContourLine, error) {
	grid, lonList := expandGridForLongitude(data)
	latList := data.LatList
	rows := len(grid)
	cols := len(grid[0])

	// 计算数值范围，并检查是否有异常值
	min, max := grid[0][0], grid[0][0]
	var validCount, nanCount, infCount int
	var sum float64

	for i := range grid {
		for j := range grid[i] {
			val := grid[i][j]
			// 检查 NaN 和 Inf
			if val != val { // NaN check
				nanCount++
				continue
			}
			if val > 1e30 || val < -1e30 { // Inf check
				infCount++
				continue
			}
			validCount++
			sum += val
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	}

	flat := make([]float64, 0, rows*cols)
	for i := 0; i < rows; i++ {
		flat = append(flat, grid[i]...)
	}

	cm := contourmap.FromFloat64s(cols, rows, flat)

	// 计算等值线值，使用 MinValue 和 MaxValue 限制范围
	var values []float64

	// 确定实际的最小值和最大值
	effectiveMin := min
	effectiveMax := max

	// 如果用户指定了 MinValue 和 MaxValue，则使用指定的范围
	if data.MinValue > 0 && data.MaxValue > 0 && data.MaxValue > data.MinValue {
		effectiveMin = data.MinValue
		effectiveMax = data.MaxValue
	}

	// 生成等值线值列表
	for v := effectiveMin; v <= effectiveMax; v += data.Step {
		values = append(values, v)
	}

	var result []ContourLine
	var totalPaths, filteredByLength, filteredByMinPoints, filteredByValue, crossBoundary int
	var lengthDistribution [10]int // 统计长度分布

	for _, v := range values {
		// 再次确认等值线值在允许范围内（双重保险）
		if data.MinValue > 0 && data.MaxValue > 0 {
			if v < data.MinValue || v > data.MaxValue {
				filteredByValue++
				continue
			}
		}

		paths := cm.Contours(v)
		totalPaths += len(paths)

		for _, path := range paths {
			// 过滤掉太短的等值线
			if len(path) < 10 { // 增加最小点数要求
				filteredByMinPoints++
				continue
			}
			line := make([]ContourPoint, len(path))
			for i, pt := range path {
				lon := interpolate(lonList, pt.X)
				lat := interpolate(latList, pt.Y)
				line[i] = ContourPoint{X: lon, Y: lat}
			}

			// 分割跨越 -180/180 边界的等值线
			splitLines := splitCrossBoundaryLine(line, lonList)
			if len(splitLines) > 1 {
				crossBoundary++
			}

			for _, subLine := range splitLines {
				// 先进行高斯平滑
				gaussianSmooth := gaussianSmoothLine(subLine, 7, 1.5)

				// 抽稀：只保留足够长的等值线
				if len(gaussianSmooth) >= 10 {
					// 使用较小的tolerance进行简化，保留更多细节
					simplified := simplifyContourLine(gaussianSmooth, 0.3)

					// 过滤掉太短的等值线（基于地理距离）
					if len(simplified) >= 5 {
						lineLength := calculateLineLength(simplified)
						// 只保留长度超过2度（约222km）的等值线
						if lineLength > 2.0 {
							// 使用 Catmull-Rom 样条进行最终平滑
							finalSmooth := catmullRomSpline(simplified, 5)

							result = append(result, ContourLine{
								Value:  v,
								Points: finalSmooth,
							})

							// 统计长度分布
							idx := int(lineLength / 10)
							if idx >= 10 {
								idx = 9
							}
							lengthDistribution[idx]++
						} else {
							filteredByLength++
						}
					}
				}
			}
		}
	}

	return result, nil
}

// 分割跨越经度边界的等值线
func splitCrossBoundaryLine(line []ContourPoint, lonList []float64) [][]ContourPoint {
	if len(line) < 2 {
		return [][]ContourPoint{line}
	}

	// 计算经度范围的中点，用于判断是否跨越边界
	lonMin := lonList[0]
	lonMax := lonList[len(lonList)-1]
	threshold := (lonMax - lonMin) * 0.5 // 如果相邻两点经度差超过一半范围，说明跨越了边界

	var result [][]ContourPoint
	var currentLine []ContourPoint

	for i := 0; i < len(line); i++ {
		// 将经度规范化到 -180 到 180 范围
		lon := line[i].X
		if lon > 180 {
			lon -= 360
		} else if lon < -180 {
			lon += 360
		}
		line[i].X = lon

		if i == 0 {
			currentLine = append(currentLine, line[i])
			continue
		}

		// 检查是否跨越边界
		lonDiff := abs(line[i].X - line[i-1].X)
		if lonDiff > threshold {
			// 跨越了边界，保存当前线段，开始新的线段
			if len(currentLine) >= 3 {
				result = append(result, currentLine)
			}
			currentLine = []ContourPoint{line[i]}
		} else {
			currentLine = append(currentLine, line[i])
		}
	}

	// 保存最后一段
	if len(currentLine) >= 3 {
		result = append(result, currentLine)
	}

	// 如果没有分割，返回原始线段
	if len(result) == 0 {
		return [][]ContourPoint{line}
	}

	return result
}

// ====== 3. 等值线平滑 ======
// 高斯平滑 - 使用高斯核进行加权平滑
func gaussianSmoothLine(line []ContourPoint, windowSize int, sigma float64) []ContourPoint {
	if len(line) <= 3 {
		return line
	}

	// 生成高斯核
	halfWindow := windowSize / 2
	kernel := make([]float64, windowSize)
	sum := 0.0
	for i := 0; i < windowSize; i++ {
		x := float64(i - halfWindow)
		kernel[i] = exp(-(x * x) / (2 * sigma * sigma))
		sum += kernel[i]
	}
	// 归一化
	for i := 0; i < windowSize; i++ {
		kernel[i] /= sum
	}

	result := make([]ContourPoint, len(line))
	for i := 0; i < len(line); i++ {
		sumX, sumY, weightSum := 0.0, 0.0, 0.0
		for j := 0; j < windowSize; j++ {
			idx := i - halfWindow + j
			if idx >= 0 && idx < len(line) {
				weight := kernel[j]
				sumX += line[idx].X * weight
				sumY += line[idx].Y * weight
				weightSum += weight
			}
		}
		result[i] = ContourPoint{
			X: sumX / weightSum,
			Y: sumY / weightSum,
		}
	}
	return result
}

// Catmull-Rom 样条插值 - 生成平滑的曲线
func catmullRomSpline(line []ContourPoint, segmentsPerPoint int) []ContourPoint {
	if len(line) < 4 {
		return line
	}

	var result []ContourPoint

	// 对于每对相邻的点，使用 Catmull-Rom 插值
	for i := 0; i < len(line)-1; i++ {
		var p0, p1, p2, p3 ContourPoint

		// 选择控制点
		if i == 0 {
			p0 = line[0]
		} else {
			p0 = line[i-1]
		}
		p1 = line[i]
		p2 = line[i+1]
		if i == len(line)-2 {
			p3 = line[len(line)-1]
		} else {
			p3 = line[i+2]
		}

		// 在 p1 和 p2 之间插值
		for j := 0; j < segmentsPerPoint; j++ {
			t := float64(j) / float64(segmentsPerPoint)
			t2 := t * t
			t3 := t2 * t

			// Catmull-Rom 公式
			x := 0.5 * ((2 * p1.X) +
				(-p0.X+p2.X)*t +
				(2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 +
				(-p0.X+3*p1.X-3*p2.X+p3.X)*t3)

			y := 0.5 * ((2 * p1.Y) +
				(-p0.Y+p2.Y)*t +
				(2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 +
				(-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3)

			result = append(result, ContourPoint{X: x, Y: y})
		}
	}

	// 添加最后一个点
	result = append(result, line[len(line)-1])

	return result
}

// Douglas-Peucker 算法简化等值线
func simplifyContourLine(line []ContourPoint, tolerance float64) []ContourPoint {
	if len(line) <= 2 {
		return line
	}

	// 找到距离起点和终点连线最远的点
	maxDist := 0.0
	maxIdx := 0
	start := line[0]
	end := line[len(line)-1]

	for i := 1; i < len(line)-1; i++ {
		dist := perpendicularDistance(line[i], start, end)
		if dist > maxDist {
			maxDist = dist
			maxIdx = i
		}
	}

	// 如果最大距离大于容差，递归简化
	if maxDist > tolerance {
		left := simplifyContourLine(line[:maxIdx+1], tolerance)
		right := simplifyContourLine(line[maxIdx:], tolerance)
		// 合并结果，去除重复的中间点
		result := append(left[:len(left)-1], right...)
		return result
	}

	// 否则返回起点和终点
	return []ContourPoint{start, end}
}

// 计算点到直线的垂直距离
func perpendicularDistance(point, lineStart, lineEnd ContourPoint) float64 {
	dx := lineEnd.X - lineStart.X
	dy := lineEnd.Y - lineStart.Y

	// 如果起点和终点重合
	if dx == 0 && dy == 0 {
		return distance(point, lineStart)
	}

	// 使用点到直线距离公式
	numerator := abs(dy*point.X - dx*point.Y + lineEnd.X*lineStart.Y - lineEnd.Y*lineStart.X)
	denominator := sqrt(dx*dx + dy*dy)
	return numerator / denominator
}

// 计算两点之间的距离
func distance(p1, p2 ContourPoint) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return sqrt(dx*dx + dy*dy)
}

// 计算等值线的总长度（简单欧氏距离，单位为度）
func calculateLineLength(line []ContourPoint) float64 {
	if len(line) < 2 {
		return 0
	}
	var totalLength float64
	for i := 1; i < len(line); i++ {
		totalLength += distance(line[i], line[i-1])
	}
	return totalLength
}

// 辅助数学函数
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	// 使用牛顿法
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// exp 函数 - 使用泰勒级数展开
func exp(x float64) float64 {
	if x < -10 {
		return 0
	}
	if x > 10 {
		return 22026.465794806718 // e^10
	}

	// 泰勒级数: e^x = 1 + x + x^2/2! + x^3/3! + ...
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
		if abs(term) < 1e-10 {
			break
		}
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
	if err := data.check(); err != nil {
		return fmt.Errorf("check data error: %v", err)
	}

	contourLines, err := generateContourLinesWithContourMap(data)
	if err != nil {
		return fmt.Errorf("generate contour lines error: %v", err)
	}
	if err := saveAsGeoJSON(contourLines, data.OutFilePath); err != nil {
		return fmt.Errorf("save geojson error: %v", err)
	}
	return nil
}
