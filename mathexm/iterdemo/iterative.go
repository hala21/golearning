package iterdemo

import "math"

func GetNumberOfWheatItera(grid int) uint64 {
	var sum, numberOfGrid uint64 = 0, 0
	numberOfGrid = 1
	sum += numberOfGrid

	for i := 2; i <= grid; i++ {
		numberOfGrid *= 2
		sum += numberOfGrid
	}

	return sum
}

func GetSquareRoot(n int, detalThreshold float64, maxTry int) float64 {
	// number: 待求的数
	// detal: 精度
	//maxtry: 最大
	if n < 1 {
		return -1.0
	}
	var minnum float64 = 1.0
	var maxnum = float64(n)
	var middlenum float64
	for i := 0; i <= maxTry; i++ {
		middlenum = (maxnum + minnum) / 2
		square := middlenum * middlenum
		detal := math.Abs(square/float64(n) - 1)
		if detal <= detalThreshold {
			return middlenum
		} else {
			if int(square) > n {
				maxnum = middlenum
			} else {
				minnum = middlenum
			}
		}

	}
	return middlenum
}
