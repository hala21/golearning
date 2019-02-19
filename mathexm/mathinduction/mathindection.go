package mathinduction

import "math"

// 证明归纳方法

type Result struct {
	wheel      int64
	wheelTotal int64
}

func (re *Result) Prove(k int) bool {
	var provePreRe = &Result{re.wheel, re.wheelTotal}
	// 证明n =1时命题是否成立
	if k == 1 {
		if int64(math.Pow(2, 1)-1) == 1 {
			re.wheel = 1
			re.wheelTotal = 1
			return true
		} else {
			return false
		}
	} else {
		//如果n=k-1时命题成立，n=k时命题是否成立
		var provePre bool = provePreRe.Prove(k - 1)
		re.wheel *= 2
		re.wheelTotal += re.wheel
		var proveCurre bool = false
		if int64(math.Pow(2, float64(k))-1) == re.wheelTotal {
			proveCurre = true
		}
		if provePre && proveCurre {
			return true
		} else {
			return false
		}

	}

}
