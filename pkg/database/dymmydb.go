package database

type dummyDatabase struct{}

func (d dummyDatabase) Put(key, value string) (string, error) {
	return "dummy put accepted", nil
}

func (d dummyDatabase) Get(key string) (string, error) {
	// todo: to have more flexible way generating returns
	return "dummy value for key " + key, nil
}

func (d dummyDatabase) Delete(key string) error {
	return nil
}

func (d dummyDatabase) Close() error {
	return nil
}

func newDummyDatabase() Database {
	return dummyDatabase{}
}
