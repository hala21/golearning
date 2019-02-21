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
