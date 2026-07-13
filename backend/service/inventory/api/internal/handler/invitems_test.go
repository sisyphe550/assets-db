package handler

import "testing"

func TestSelectExpectedTaskAssetsFiltersConfiguredItems(t *testing.T) {
	scopeAssets := []map[string]any{
		{"id": float64(1), "asset_no": "EQUIP-1", "name": "A", "location": "L1"},
		{"id": float64(2), "asset_no": "EQUIP-2", "name": "B", "location": "L2"},
		{"id": float64(3), "asset_no": "EQUIP-3", "name": "C", "location": "L3"},
	}

	got := selectExpectedTaskAssets(scopeAssets, []int64{2, 3}, false)

	if len(got) != 2 {
		t.Fatalf("expected 2 selected assets, got %d: %#v", len(got), got)
	}
	if rpcAssetNo(got[0]) != "EQUIP-2" || rpcAssetNo(got[1]) != "EQUIP-3" {
		t.Fatalf("unexpected selection order: %#v", got)
	}
}

func TestSelectExpectedTaskAssetsKeepsLegacyFallback(t *testing.T) {
	scopeAssets := []map[string]any{
		{"id": float64(1), "asset_no": "EQUIP-1"},
		{"id": float64(2), "asset_no": "EQUIP-2"},
	}

	got := selectExpectedTaskAssets(scopeAssets, nil, true)

	if len(got) != len(scopeAssets) {
		t.Fatalf("legacy task without configured items must keep scope assets, got %#v", got)
	}
}
