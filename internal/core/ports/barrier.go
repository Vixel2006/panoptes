package port

type BarrierPort interface {
	Lock()
	Unlock()
	Decision() bool
	Release(forward bool)
	SetActive(active bool)
}
