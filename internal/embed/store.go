package embed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	chromem "github.com/philippgille/chromem-go"
)

type Store struct {
	collection *chromem.Collection
	dir        string
}

func NewStore(brainDir string) (*Store, error) {
	dir := filepath.Join(brainDir, "vectors")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "vectors.cbdb")
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open vector store: %w", err)
	}

	collection := db.GetCollection("entries", nil)
	if collection == nil {
		collection, err = db.CreateCollection("entries", nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return &Store{
		collection: collection,
		dir:        dir,
	}, nil
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) IndexEntries(entries []IndexEntry, provider Provider) error {
	if _, ok := provider.(*NoneProvider); ok {
		return fmt.Errorf("embedding not configured")
	}

	ctx := context.Background()
	ids := make([]string, 0, len(entries))
	embeddings := make([][]float32, 0, len(entries))
	documents := make([]string, 0, len(entries))
	metadatas := make([]map[string]string, 0, len(entries))

	for _, e := range entries {
		embedding, err := provider.Embed(e.Message)
		if err != nil {
			return err
		}
		ids = append(ids, e.Key)
		embeddings = append(embeddings, embedding)
		documents = append(documents, e.Message)
		metadatas = append(metadatas, map[string]string{
			"topic":     e.Topic,
			"timestamp": e.Timestamp,
		})
	}

	return s.collection.Add(ctx, ids, embeddings, metadatas, documents)
}

const retrievalInstruction = "Given the current coding context, retrieve past knowledge entries that would help avoid mistakes, apply patterns, or understand decisions relevant to this work. "

const maxEmbeddingQueryLen = 8000

func (s *Store) Search(query string, provider Provider, topK int) ([]SearchResult, error) {
	if _, ok := provider.(*NoneProvider); ok {
		return nil, fmt.Errorf("embedding not configured")
	}

	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}

	if len(query) > maxEmbeddingQueryLen {
		query = query[:maxEmbeddingQueryLen]
	}

	instructionQuery := retrievalInstruction + query
	embedding, err := provider.Embed(instructionQuery)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	results, err := s.collection.QueryEmbedding(ctx, embedding, topK, nil, nil)
	if err != nil {
		return nil, err
	}

	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			Key:   r.ID,
			Topic: r.Metadata["topic"],
			Msg:   r.Content,
			Score: r.Similarity,
		})
	}

	return searchResults, nil
}

type SearchResult struct {
	Key   string
	Topic string
	Msg   string
	Score float32
}

type IndexEntry struct {
	Key       string
	Topic     string
	Message   string
	Timestamp string
}
