package utility

import "math"

func BinarySearch(arr []string, target string) int {
	left, right := 0, len(arr)-1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return mid
		}

		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1
}

func Dijkstra(graph [][]int, start int) []int {
	numNodes := len(graph)
	dist := make([]int, numNodes)
	visited := make([]bool, numNodes)

	for i := range dist {
		dist[i] = math.MaxInt32
	}

	dist[start] = 0

	for count := 0; count < numNodes-1; count++ {
		u := MinDistance(dist, visited)

		visited[u] = true

		for v := 0; v < numNodes; v++ {
			if !visited[v] && graph[u][v] != 0 && dist[u] != math.MaxInt32 && dist[u]+graph[u][v] < dist[v] {
				dist[v] = dist[u] + graph[u][v]
			}
		}
	}

	return dist
}

func MinDistance(dist []int, visited []bool) int {
	min := math.MaxInt32
	minIndex := -1

	for v := range dist {
		if !visited[v] && dist[v] <= min {
			min = dist[v]
			minIndex = v
		}
	}

	return minIndex
}

func QuickSort(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}

	pivot := arr[0]
	var left, right []int

	for _, value := range arr[1:] {
		if value <= pivot {
			left = append(left, value)
		} else {
			right = append(right, value)
		}
	}

	left = QuickSort(left)
	right = QuickSort(right)

	return append(append(left, pivot), right...)
}
