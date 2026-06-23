package jwf

func (s *WorkFlow) init() error {
	if s.IsInitial {
		return nil
	}

	// Manual Trigger
	s.newStep(KindTrigger, "step:manual:trigger", "manual:trigger", "1.0.0", "Manual Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Webhook Trigger
	s.newStep(KindTrigger, "step:webhook:trigger", "webhook:trigger", "1.0.0", "Webhook Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Schedule Trigger
	s.newStep(KindTrigger, "step:schedule:trigger", "schedule:trigger", "1.0.0", "Schedule Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Condition
	s.newStep(KindCondition, "step:condition", "condition", "1.0.0", "Condition", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Switch
	s.newStep(KindCondition, "step:switch", "switch", "1.0.0", "Switch", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Wait
	s.newStep(KindDelay, "step:delay", "delay", "1.0.0", "Wait", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// For loop
	s.newStep(KindBucle, "step:bucle", "bucle", "1.0.0", "For loop", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	// Action
	s.newStep(KindAction, "step:action", "action", "1.0.0", "Action", "root").
		setDefinition(``).
		setOnPublish(``).
		save()

	s.IsInitial = true
	return nil
}
