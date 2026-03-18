package repository

import (
	"testing"
	"time"

	"github.com/usedcvnt/microtwitter/comment-svc/internal/model"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestBuildTree_FlatListNestedTree(t *testing.T) {
	roots := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root1"},
		{ID: "2", PostID: "p1", UserID: "u2", Content: "root2"},
	}
	all := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root1"},
		{ID: "2", PostID: "p1", UserID: "u2", Content: "root2"},
		{ID: "3", PostID: "p1", UserID: "u3", ParentID: strPtr("1"), Content: "child of root1"},
		{ID: "4", PostID: "p1", UserID: "u4", ParentID: strPtr("2"), Content: "child of root2"},
		{ID: "5", PostID: "p1", UserID: "u5", ParentID: strPtr("1"), Content: "another child of root1"},
	}

	result := buildTree(roots, all)

	if len(result) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(result))
	}

	// root1 should have 2 children
	if len(result[0].Children) != 2 {
		t.Fatalf("expected root1 to have 2 children, got %d", len(result[0].Children))
	}
	if result[0].Children[0].Content != "child of root1" {
		t.Errorf("expected first child content 'child of root1', got %q", result[0].Children[0].Content)
	}
	if result[0].Children[1].Content != "another child of root1" {
		t.Errorf("expected second child content 'another child of root1', got %q", result[0].Children[1].Content)
	}

	// root2 should have 1 child
	if len(result[1].Children) != 1 {
		t.Fatalf("expected root2 to have 1 child, got %d", len(result[1].Children))
	}
	if result[1].Children[0].Content != "child of root2" {
		t.Errorf("expected child content 'child of root2', got %q", result[1].Children[0].Content)
	}
}

func TestBuildTree_DeepNesting(t *testing.T) {
	roots := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root"},
	}
	all := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root"},
		{ID: "2", PostID: "p1", UserID: "u2", ParentID: strPtr("1"), Content: "child"},
		{ID: "3", PostID: "p1", UserID: "u3", ParentID: strPtr("2"), Content: "grandchild"},
	}

	result := buildTree(roots, all)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}
	if len(result[0].Children) != 1 {
		t.Fatalf("expected root to have 1 child, got %d", len(result[0].Children))
	}
	child := result[0].Children[0]
	if child.Content != "child" {
		t.Errorf("expected child content 'child', got %q", child.Content)
	}
	if len(child.Children) != 1 {
		t.Fatalf("expected child to have 1 grandchild, got %d", len(child.Children))
	}
	grandchild := child.Children[0]
	if grandchild.Content != "grandchild" {
		t.Errorf("expected grandchild content 'grandchild', got %q", grandchild.Content)
	}
	if len(grandchild.Children) != 0 {
		t.Errorf("expected grandchild to have no children, got %d", len(grandchild.Children))
	}
}

func TestFilterDeleted_DeletedNodeWithChildren(t *testing.T) {
	now := time.Now()
	child := &model.Comment{
		ID: "2", PostID: "p1", UserID: "u2", ParentID: strPtr("1"),
		Content: "child comment",
	}
	node := &model.Comment{
		ID: "1", PostID: "p1", UserID: "u1",
		Content:   "original content",
		DeletedAt: timePtr(now),
		Children:  []*model.Comment{child},
	}

	result := filterDeleted(node)

	if result == nil {
		t.Fatal("expected deleted node with children to be preserved, got nil")
	}
	if result.Content != "[удалено]" {
		t.Errorf("expected content '[удалено]', got %q", result.Content)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child preserved, got %d", len(result.Children))
	}
	if result.Children[0].Content != "child comment" {
		t.Errorf("expected child content 'child comment', got %q", result.Children[0].Content)
	}
}

func TestFilterDeleted_DeletedNodeWithoutChildren(t *testing.T) {
	now := time.Now()
	node := &model.Comment{
		ID: "1", PostID: "p1", UserID: "u1",
		Content:   "deleted leaf",
		DeletedAt: timePtr(now),
		Children:  nil,
	}

	result := filterDeleted(node)

	if result != nil {
		t.Errorf("expected deleted node without children to be removed (nil), got %+v", result)
	}
}

func TestFilterDeleted_DeletedRootWithChildren(t *testing.T) {
	now := time.Now()
	grandchild := &model.Comment{
		ID: "3", PostID: "p1", UserID: "u3", ParentID: strPtr("2"),
		Content: "grandchild comment",
	}
	child := &model.Comment{
		ID: "2", PostID: "p1", UserID: "u2", ParentID: strPtr("1"),
		Content:  "child comment",
		Children: []*model.Comment{grandchild},
	}
	root := &model.Comment{
		ID: "1", PostID: "p1", UserID: "u1",
		Content:   "root content",
		DeletedAt: timePtr(now),
		Children:  []*model.Comment{child},
	}

	result := filterDeleted(root)

	if result == nil {
		t.Fatal("expected deleted root with children to be preserved, got nil")
	}
	if result.Content != "[удалено]" {
		t.Errorf("expected root content '[удалено]', got %q", result.Content)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	if result.Children[0].Content != "child comment" {
		t.Errorf("expected child content 'child comment', got %q", result.Children[0].Content)
	}
	if len(result.Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(result.Children[0].Children))
	}
	if result.Children[0].Children[0].Content != "grandchild comment" {
		t.Errorf("expected grandchild content 'grandchild comment', got %q", result.Children[0].Children[0].Content)
	}
}

func TestBuildTree_EmptyInputs(t *testing.T) {
	result := buildTree(nil, nil)

	if len(result) != 0 {
		t.Errorf("expected empty result for nil inputs, got %d items", len(result))
	}

	result = buildTree([]model.Comment{}, []model.Comment{})

	if len(result) != 0 {
		t.Errorf("expected empty result for empty slices, got %d items", len(result))
	}
}

func TestBuildTree_RootsOnlyNoChildren(t *testing.T) {
	roots := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root1"},
		{ID: "2", PostID: "p1", UserID: "u2", Content: "root2"},
		{ID: "3", PostID: "p1", UserID: "u3", Content: "root3"},
	}
	all := []model.Comment{
		{ID: "1", PostID: "p1", UserID: "u1", Content: "root1"},
		{ID: "2", PostID: "p1", UserID: "u2", Content: "root2"},
		{ID: "3", PostID: "p1", UserID: "u3", Content: "root3"},
	}

	result := buildTree(roots, all)

	if len(result) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(result))
	}
	for i, r := range result {
		if len(r.Children) != 0 {
			t.Errorf("expected root %d to have no children, got %d", i, len(r.Children))
		}
	}
	if result[0].Content != "root1" {
		t.Errorf("expected 'root1', got %q", result[0].Content)
	}
	if result[1].Content != "root2" {
		t.Errorf("expected 'root2', got %q", result[1].Content)
	}
	if result[2].Content != "root3" {
		t.Errorf("expected 'root3', got %q", result[2].Content)
	}
}
