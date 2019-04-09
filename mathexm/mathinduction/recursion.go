package mathinduction

import "fmt"

// 递归就是将复杂的问题，每次解决一点点，并将剩下的任务转化为更简单的问题等待下一次求解，
// 如此反复，直到最简单的形式。
var rewards = []int{1, 2, 5, 10}

func GetRewardsRecursion1(totalReward int, result []int) {

	if totalReward == 0 {
		fmt.Println(result)
		return
	} else if totalReward < 0 {
		return
	} else {
		for i := 0; i < len(rewards); i++ {
			newResult := append(result, rewards[i])
			GetRewardsRecursion1(totalReward-rewards[i], newResult)
		}
	}

}

// Merge Sort
func MergeSort(sort []int) []int {

	if len(sort) == 0 {
		return []int{0}
	}
	if len(sort) == 1 {
		return sort
	}
	middle := len(sort) / 2
	left := sort[0:middle]
	right := sort[middle:len(sort)]
	leftArray := MergeSort(left)
	rightArray := MergeSort(right)

	var merger []int = Merge(leftArray, rightArray)
	return merger
}

func Merge(left []int, right []int) []int {
	mergerOne := make([]int, len(left)+len(right))
	var ai, bi, mi = 0, 0, 0
	for ai < len(left) && bi < len(right) {
		if left[ai] < right[bi] {
			mergerOne[mi] = left[ai]
			ai++
		} else {
			mergerOne[mi] = right[bi]
			bi++
		}
		mi++
	}

	if ai < len(left) {
		for i := ai; i < len(left); i++ {
			mergerOne[mi] = left[i]
			mi++
		}
	} else {
		for i := bi; i < len(right); i++ {
			mergerOne[mi] = right[i]
			mi++
		}
	}
	return mergerOne
}
