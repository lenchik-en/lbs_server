package geo

import "math"

const earthRadiusM = 6_371_000

// HaversineDistance возвращает расстояние в метрах между двумя точками (lat/lon в градусах).
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1R := lat1 * math.Pi / 180
	lat2R := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1R)*math.Cos(lat2R)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM * c
}

// ChordAccuracy вычисляет accuracy (в метрах) по алгоритму хорды пересечения окружностей.
//
// d  — расстояние между центрами (метры)
// r1 — радиус первой окружности (accuracy источника 1)
// r2 — радиус второй окружности (accuracy источника 2)
// minAccuracy — минимально допустимая accuracy (floor), обычно 1м
//
// Алгоритм:
//  1. Строим окружность с радиусом accuracy для каждой точки
//  2. Находим их пересечение
//  3. Берём полухорду (h) пересечения как итоговую accuracy
//
// Краевые случаи:
//   - d == 0: accuracy = min(r1, r2)
//   - d < |r1-r2| (одна внутри другой): accuracy = min(r1, r2)
//   - d > r1+r2 (не пересекаются): accuracy = min(r1, r2)
//   - d == r1+r2 (касание): h → 0, применяется minAccuracy floor
func ChordAccuracy(d, r1, r2, minAccuracy float64) float64 {
	// Концентрические или одна окружность внутри другой
	if d <= math.Abs(r1-r2) {
		return math.Max(math.Min(r1, r2), minAccuracy)
	}

	// Окружности не пересекаются
	if d > r1+r2 {
		return math.Max(math.Min(r1, r2), minAccuracy)
	}

	// Нормальное пересечение: вычисляем полухорду (радиус окружности - наша точность)
	x := (d*d + r1*r1 - r2*r2) / (2 * d)
	h := math.Sqrt(math.Max(0, r1*r1-x*x))

	return math.Max(h, minAccuracy)
}
