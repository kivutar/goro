package res

import "testing"

func TestParseClientInfo(t *testing.T) {
	info, err := ParseClientInfo([]byte(`
<clientinfo>
  <servicetype>korea</servicetype>
  <connection>
    <display>TestRO</display>
    <address>127.0.0.1</address>
    <port>6900</port>
    <version>55</version>
    <langtype>0</langtype>
    <aid><admin>2000002</admin></aid>
    <yellow>
      <admin>2000003</admin>
      <admin>2000002</admin>
      <admin>not-a-number</admin>
    </yellow>
  </connection>
</clientinfo>`))
	if err != nil {
		t.Fatal(err)
	}
	if info.ServiceType != "korea" {
		t.Fatalf("service type = %q", info.ServiceType)
	}
	if len(info.Connections) != 1 {
		t.Fatalf("connections = %d", len(info.Connections))
	}
	conn := info.Connections[0]
	if conn.Display != "TestRO" || conn.Address != "127.0.0.1" || conn.Port != 6900 || conn.Version != 55 {
		t.Fatalf("unexpected connection: %+v", conn)
	}
	wantAdmins := []uint32{2000003, 2000002}
	if len(conn.AdminList) != len(wantAdmins) {
		t.Fatalf("admin list = %v, want %v", conn.AdminList, wantAdmins)
	}
	for i := range wantAdmins {
		if conn.AdminList[i] != wantAdmins[i] {
			t.Fatalf("admin list = %v, want %v", conn.AdminList, wantAdmins)
		}
	}
}

func TestClientInfoServiceDefaultsToKorea(t *testing.T) {
	for raw, want := range map[string]string{"": "korea", "america": "america", "Malaysia": "korea", "bogus": "korea", "singapore": "singapore"} {
		if got := (ClientInfo{ServiceType: raw}).Service(); got != want {
			t.Fatalf("Service(%q) = %q, want %q", raw, got, want)
		}
	}
}
