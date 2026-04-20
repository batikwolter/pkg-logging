package observe

// Resource identifies service identity metadata.
type Resource struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
}

func (r Resource) fields() []Field {
	fields := []Field{
		String("service.name", r.ServiceName),
		String("env", r.Environment),
	}
	if r.ServiceVersion != "" {
		fields = append(fields, String("service.version", r.ServiceVersion))
	}
	return fields
}
