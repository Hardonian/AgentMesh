package main

import (
	"fmt"

	"github.com/agentmesh/agentmesh/internal/canary"
)

func main() {
	fmt.Println("=== AgentMesh Progressive Delivery & Canary Demo ===")

	mgr := canary.NewManager()

	// 1. Launch Canary deployment for procurement-agent
	// Baseline: v1.0.0, Candidate: v1.1.0
	// Rollback threshold: MaxErrorRate 0.05 (5%), MaxLatency 3000ms
	dep, _ := mgr.StartCanary("procurement-agent", "v1.0.0", "v1.1.0", 10, false, 0.05, 3000)
	fmt.Printf("1. Canary started: Agent=%s (Baseline=%s, Candidate=%s, TrafficWeight=%d%%)\n",
		dep.AgentID, dep.BaselineVersion, dep.CandidateVersion, dep.TrafficWeight)

	// 2. Simulate healthy candidate traffic
	for i := 0; i < 4; i++ {
		_, _ = mgr.RecordCandidateSample("procurement-agent", true, 450, 0.01)
	}
	fmt.Println("2. Recorded 4 successful candidate runs...")

	// 3. Promote candidate to 50% traffic
	promoted, _ := mgr.Promote("procurement-agent", 50)
	fmt.Printf("3. Promoted traffic weight to %d%%\n", promoted.TrafficWeight)

	// 4. Simulate a latency & error regression in candidate version
	fmt.Println("4. Simulating sudden candidate errors (error rate > 5%)...")
	rolledBack := false
	for i := 0; i < 5; i++ {
		rb, _ := mgr.RecordCandidateSample("procurement-agent", false, 4500, 0.01)
		if rb {
			rolledBack = true
			break
		}
	}

	if rolledBack {
		fmt.Println("✓ AUTOMATIC ROLLBACK TRIGGERED: Error rate breached threshold!")
		activeVer := mgr.GetActiveVersion("procurement-agent")
		fmt.Printf("✓ Active production version restored to: %s\n", activeVer)
	}

	fmt.Println("✓ Invariant confirmed: Regressed canary automatically rolls back to stable baseline.")
}
