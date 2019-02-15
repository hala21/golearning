package iterdemo

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

/**
func main(){
	wheats := getNumberOfWheatItera(2)
	print(wheats)
}
**/
