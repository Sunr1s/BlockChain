package blockchain

// MemPool represents a very simple memory pool for unconfirmed transactions
// There is an expiration period for each TX in real implementations.
//
// You can read more about mempool:
// https://bitcoin.stackexchange.com/questions/46152/how-do-transactions-leave-the-memory-pool
// https://bitcoin.stackexchange.com/questions/41536/why-does-bitcoin-keep-transactions-in-a-memory-pool

// New returns new MemPool
func NewPool() *MemPool {
	return &MemPool{
		pool: make(map[string]Transaction, 1024),
	}
}

// Add adds new transaction to the pool
func (m *MemPool) Add(tx *Transaction) {
	m.l.Lock()
	m.pool[string(tx.CurrHash)] = *tx
	m.l.Unlock()
}

// GetByID returns a transaction with a given transaction hex-encoded ID
func (m *MemPool) GetByID(CurrHash []byte) *Transaction {
	m.l.RLock()
	defer m.l.RUnlock()

	tx, ok := m.pool[string(CurrHash)]
	if !ok {
		return nil
	}
	return &tx
}

// Get returns N transactions and cleans the pool
func (m *MemPool) Get(n int) []Transaction {
	foundTXs := make([]Transaction, 0, n)
	m.l.Lock()
	defer func() {
		for _, tx := range foundTXs {
			_tx := string(tx.CurrHash)
			delete(m.pool, _tx)
		}
		m.l.Unlock()
	}()

	for _, t := range m.pool {
		if len(foundTXs) >= n {
			return foundTXs
		}
		foundTXs = append(foundTXs, t)
	}
	return foundTXs
}

// Read returns N transactions without cleaning
func (m *MemPool) Read(n int) []Transaction {
	m.l.RLock()
	defer m.l.RUnlock()

	txs := make([]Transaction, 0, n)
	for _, t := range m.pool {
		txs = append(txs, t)
	}
	return txs
}

// DeleteByID removes a transaction with a given transaction hex-encoded ID
func (m *MemPool) DeleteByID(txHexID string) {
	m.l.Lock()
	defer m.l.Unlock()
	delete(m.pool, txHexID)
}

// Size return size of the MemPool
func (m *MemPool) Size() int {
	return len(m.pool)
}
