# LFU cache
My LFU cache implementation

## Operations 
* `Get(key K) (V, error)`
* `Put(key K, value V)`
* `All() iter.Seq2[K, V]`
* `Size() int`
* `Capacity() int`
* `GetKeyFrequency(key K) (int, error)`

