package ticket

import "time"

// CheckDependencies checks if all dependencies of a ticket are resolved (DONE or CANCELLED).
// Returns the list of unresolved dependency IDs.
func CheckDependencies(store *Store, tk *Ticket) ([]string, error) {
	var unresolved []string
	for _, depID := range tk.DependsOn {
		dep, err := store.Get(depID)
		if err != nil {
			// If we can't find the dependency, treat it as unresolved.
			unresolved = append(unresolved, depID)
			continue
		}
		if dep.Status != StatusDone && dep.Status != StatusCancelled {
			unresolved = append(unresolved, depID)
		}
	}
	return unresolved, nil
}

// FindDependents returns all tickets that depend on the given ticket ID.
// Scans all tickets across all projects (cross-project).
func FindDependents(store *Store, ticketID string) ([]*Ticket, error) {
	all, err := store.List(ListFilter{})
	if err != nil {
		return nil, err
	}

	var dependents []*Ticket
	for _, tk := range all {
		for _, dep := range tk.DependsOn {
			if dep == ticketID {
				dependents = append(dependents, tk)
				break
			}
		}
	}
	return dependents, nil
}

// AutoUnblock checks dependents of a completed ticket and unblocks them
// if all their dependencies are now resolved. Returns the list of unblocked tickets.
func AutoUnblock(store *Store, ticketID string) ([]*Ticket, error) {
	dependents, err := FindDependents(store, ticketID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var unblocked []*Ticket

	for _, tk := range dependents {
		if tk.Status != StatusBlocked || tk.PriorStatus == nil {
			continue
		}

		unresolved, err := CheckDependencies(store, tk)
		if err != nil {
			continue
		}
		if len(unresolved) > 0 {
			continue
		}

		// All dependencies resolved — snap back to prior status.
		tk.Status = *tk.PriorStatus
		tk.PriorStatus = nil
		tk.Updated = now

		AppendSection(tk, "Auto-Unblocked", "st", "", "", nil, now)

		if err := store.Save(tk); err != nil {
			continue
		}

		unblocked = append(unblocked, tk)
	}

	return unblocked, nil
}
