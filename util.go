package lightstep

func IsMetaSpan(s *spanImpl) bool {
	return s.raw.Tags["lightstep.meta_event"] != nil
}
