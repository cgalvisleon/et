package jwf

func (s *WorkFlow) init() error {
	if s.isInitial {
		return nil
	}

	// Manual Trigger
	err := s.newStep(KindTrigger, "step:manual:trigger", "manual:trigger", "1.0.0", "Manual Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Webhook Trigger
	err = s.newStep(KindTrigger, "step:webhook:trigger", "webhook:trigger", "1.0.0", "Webhook Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Schedule Trigger
	err = s.newStep(KindTrigger, "step:schedule:trigger", "schedule:trigger", "1.0.0", "Schedule Trigger", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Condition
	err = s.newStep(KindCondition, "step:condition", "condition", "1.0.0", "Condition", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Switch
	err = s.newStep(KindCondition, "step:switch", "switch", "1.0.0", "Switch", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Wait
	err = s.newStep(KindDelay, "step:delay", "delay", "1.0.0", "Wait", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// For loop
	err = s.newStep(KindBucle, "step:bucle", "bucle", "1.0.0", "For loop", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	// Action
	err = s.newStep(KindAction, "step:action", "action", "1.0.0", "Action", "root").
		setDefinition(``).
		setOnPublish(``).
		save()
	if err != nil {
		return err
	}

	s.isInitial = true
	return nil
}
