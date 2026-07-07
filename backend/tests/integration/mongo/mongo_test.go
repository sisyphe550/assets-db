//go:build integration
// +build integration

package mongotest

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoInventoryDraft(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("MongoDB ping failed: %v", err)
	}

	col := client.Database("fams_inventory").Collection("inventory_draft")

	// Test 1: Insert draft
	draft := bson.M{
		"task_id":     70001,
		"asset_no":    "EQUIP-2026-0091",
		"operator_id": 10003,
		"modified_cells": bson.M{
			"actual_location": "一号实验楼302",
			"temp_notes":      "略有磨损",
		},
		"updated_at": time.Now(),
	}
	_, err = col.InsertOne(ctx, draft)
	if err != nil {
		t.Fatal("insert:", err)
	}
	t.Log("draft inserted")

	// Test 2: Find draft
	var result bson.M
	err = col.FindOne(ctx, bson.M{"task_id": 70001, "asset_no": "EQUIP-2026-0091"}).Decode(&result)
	if err != nil {
		t.Fatal("find:", err)
	}
	if result["operator_id"] != int32(10003) {
		t.Errorf("unexpected operator_id: %v", result["operator_id"])
	}
	t.Log("draft found")

	// Test 3: Duplicate key (should fail)
	_, err = col.InsertOne(ctx, draft)
	if err == nil {
		t.Error("duplicate insert should have failed")
	}
	t.Log("duplicate prevented:", err)

	// Test 4: Cleanup
	col.DeleteOne(ctx, bson.M{"task_id": 70001, "asset_no": "EQUIP-2026-0091"})
	t.Log("cleanup done")
}
