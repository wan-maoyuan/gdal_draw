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

	// 打印数据统计信息用于调试
	fmt.Printf("数据范围: rows=%d, cols=%d\n", rows, cols)
	fmt.Printf("纬度范围: [%.2f, %.2f]\n", latList[0], latList[len(latList)-1])
	fmt.Printf("经度范围: [%.2f, %.2f]\n", lonList[0], lonList[len(lonList)-1])

	// 计算数值范围，并检查是否有异常值
	min, max := grid[0][0], grid[0][0]
	var validCount, nanCount, infCount int
	var sum float64

	// 采样一些点的值用于调试
	fmt.Println("采样数据点（纬度, 经度, 值）:")
	samplePoints := [][2]int{
		{rows / 4, cols / 4},         // 第一象限
		{rows / 4, 3 * cols / 4},     // 第二象限
		{3 * rows / 4, cols / 4},     // 第三象限
		{3 * rows / 4, 3 * cols / 4}, // 第四象限
		{rows / 2, cols / 2},         // 中心点（赤道）
		{0, cols / 2},                // 北极点
		{rows - 1, cols / 2},         // 南极点
	}
	for _, pt := range samplePoints {
		if pt[0] < rows && pt[1] < cols {
			lat := latList[pt[0]]
			lon := lonList[pt[1]]
			val := grid[pt[0]][pt[1]]
			fmt.Printf("  (%.2f, %.2f) = %.2f\n", lat, lon, val)
		}
	}

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
	avg := sum / float64(validCount)
	fmt.Printf("数值统计: min=%.2f, max=%.2f, avg=%.2f, valid=%d, NaN=%d, Inf=%d\n",
		min, max, avg, validCount, nanCount, infCount)

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
		fmt.Printf("使用指定的值范围: [%.2f, %.2f] (数据实际范围: [%.2f, %.2f])\n",
			effectiveMin, effectiveMax, min, max)
	} else {
		fmt.Printf("使用数据实际范围: [%.2f, %.2f]\n", min, max)
	}

	// 生成等值线值列表
	for v := effectiveMin; v <= effectiveMax; v += data.Step {
		values = append(values, v)
	}
	fmt.Printf("将生成 %d 条等值线，步长=%.2f\n", len(values), data.Step)

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
				// 使用自适应的平滑窗口
				windowSize := 3
				if len(subLine) > 20 {
					windowSize = 5
				}
				smooth := smoothContourLine(subLine, windowSize)
				// 抽稀：只保留足够长的等值线
				if len(smooth) >= 8 { // 增加要求
					// 简化等值线，减少点数
					simplified := simplifyContourLine(smooth, 0.8) // 进一步增大tolerance

					// 过滤掉太短的等值线（基于地理距离）
					if len(simplified) >= 5 {
						lineLength := calculateLineLength(simplified)
						// 只保留长度超过2度（约222km）的等值线
						if lineLength > 2.0 {
							result = append(result, ContourLine{
								Value:  v,
								Points: simplified,
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

	for i := 0; i < 10; i++ {
		if lengthDistribution[i] > 0 {
			fmt.Printf("  %d-%d: %d条\n", i*10, (i+1)*10, lengthDistribution[i])
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
