package telemetry

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// SocketFlowMetric records socket-level transport telemetry between agents and tools.
type SocketFlowMetric struct {
	FlowKey            string    `json:"flowKey"`
	SourceAgent        string    `json:"sourceAgent"`
	DestAgent          string    `json:"destAgent"`
	Protocol           string    `json:"protocol"` // "A2A", "MCP", "HTTP"
	BytesTransferred   int64     `json:"bytesTransferred"`
	PacketsTotal       int64     `json:"packetsTotal"`
	PacketsDropped     int64     `json:"packetsDropped"`
	RoundTripLatencyMs float64   `json:"roundTripLatencyMs"`
	LastActive         time.Time `json:"lastActive"`
}

// AgentNetworkSummary aggregates socket health for an individual agent.
type AgentNetworkSummary struct {
	AgentID              string  `json:"agentId"`
	TotalInboundBytes    int64   `json:"totalInboundBytes"`
	TotalOutboundBytes   int64   `json:"totalOutboundBytes"`
	ActivePeerCount      int     `json:"activePeerCount"`
	AverageSocketLatency float64 `json:"averageSocketLatencyMs"`
	PacketDropRate       float64 `json:"packetDropRate"`
	IsEBPFActive         bool    `json:"isEbpfActive"`
}

// EBPFOpsObserver monitors high-throughput agent networks via eBPF kernel hooks or userspace socket probes.
type EBPFOpsObserver struct {
	mu           sync.RWMutex
	flows        map[string]*SocketFlowMetric
	isLinuxKernel bool
}

// NewEBPFOpsObserver initializes the network observer.
func NewEBPFOpsObserver() *EBPFOpsObserver {
	isLinux := runtime.GOOS == "linux"
	return &EBPFOpsObserver{
		flows:        make(map[string]*SocketFlowMetric),
		isLinuxKernel: isLinux,
	}
}

// RecordSocketActivity ingests telemetry from eBPF socket maps or userspace connection hooks.
func (o *EBPFOpsObserver) RecordSocketActivity(source, dest, protocol string, bytes int64, latencyMs float64, droppedPackets int64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	flowKey := fmt.Sprintf("%s->%s:%s", source, dest, protocol)
	metric, ok := o.flows[flowKey]
	if !ok {
		metric = &SocketFlowMetric{
			FlowKey:     flowKey,
			SourceAgent: source,
			DestAgent:   dest,
			Protocol:    protocol,
		}
		o.flows[flowKey] = metric
	}

	metric.BytesTransferred += bytes
	metric.PacketsTotal++
	metric.PacketsDropped += droppedPackets
	metric.LastActive = time.Now().UTC()

	// Rolling average for round-trip latency
	if metric.RoundTripLatencyMs == 0 {
		metric.RoundTripLatencyMs = latencyMs
	} else {
		metric.RoundTripLatencyMs = (metric.RoundTripLatencyMs * 0.8) + (latencyMs * 0.2)
	}
}

// GetFlows returns all monitored socket flows.
func (o *EBPFOpsObserver) GetFlows() []SocketFlowMetric {
	o.mu.RLock()
	defer o.mu.RUnlock()

	out := make([]SocketFlowMetric, 0, len(o.flows))
	for _, f := range o.flows {
		out = append(out, *f)
	}
	return out
}

// GetAgentSummary aggregates all flows involving a specific agent.
func (o *EBPFOpsObserver) GetAgentSummary(agentID string) *AgentNetworkSummary {
	o.mu.RLock()
	defer o.mu.RUnlock()

	summary := &AgentNetworkSummary{
		AgentID:      agentID,
		IsEBPFActive: o.isLinuxKernel,
	}

	peers := make(map[string]bool)
	var totalLatency float64
	var flowCount int64
	var totalPackets int64
	var totalDropped int64

	for _, f := range o.flows {
		if f.SourceAgent == agentID {
			summary.TotalOutboundBytes += f.BytesTransferred
			peers[f.DestAgent] = true
			totalLatency += f.RoundTripLatencyMs
			flowCount++
			totalPackets += f.PacketsTotal
			totalDropped += f.PacketsDropped
		} else if f.DestAgent == agentID {
			summary.TotalInboundBytes += f.BytesTransferred
			peers[f.SourceAgent] = true
			totalLatency += f.RoundTripLatencyMs
			flowCount++
			totalPackets += f.PacketsTotal
			totalDropped += f.PacketsDropped
		}
	}

	summary.ActivePeerCount = len(peers)
	if flowCount > 0 {
		summary.AverageSocketLatency = totalLatency / float64(flowCount)
	}
	if totalPackets > 0 {
		summary.PacketDropRate = float64(totalDropped) / float64(totalPackets)
	}

	return summary
}
