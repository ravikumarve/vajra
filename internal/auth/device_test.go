package auth

import (
	"testing"
)

func TestDeviceFlowApprove(t *testing.T) {
	m := NewDeviceFlowManager("http://localhost:9735")
	st := m.StartFlow()
	if st.DeviceCode == "" || st.UserCode == "" {
		t.Fatalf("expected codes, got %+v", st)
	}
	if err := m.ApproveFlow(st.UserCode, "agent-1"); err != nil {
		t.Fatalf("ApproveFlow: %v", err)
	}
	got, err := m.CheckFlow(st.DeviceCode)
	if err != nil {
		t.Fatalf("CheckFlow: %v", err)
	}
	if got.Status != "approved" || got.AgentID != "agent-1" {
		t.Fatalf("unexpected state: %+v", got)
	}
	m.RemoveFlow(st.DeviceCode)
	if _, err := m.CheckFlow(st.DeviceCode); err == nil {
		t.Fatal("expected error after remove, got nil")
	}
}

func TestDeviceFlowInvalidCodes(t *testing.T) {
	m := NewDeviceFlowManager("http://localhost:9735")
	if err := m.ApproveFlow("NOPE-0000", "agent-1"); err == nil {
		t.Fatal("expected error for unknown user code")
	}
	if _, err := m.CheckFlow("nope"); err == nil {
		t.Fatal("expected error for unknown device code")
	}
}
