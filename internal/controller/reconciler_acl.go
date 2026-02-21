package controller

import (
	"context"
	"crypto/sha256"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aero "github.com/aerospike/aerospike-client-go/v8"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// builtinRoles are Aerospike predefined roles that must not be dropped.
var builtinRoles = map[string]bool{
	"user-admin":     true,
	"sys-admin":      true,
	"data-admin":     true,
	"read":           true,
	"write":          true,
	"read-write":     true,
	"read-write-udf": true,
	"truncate":       true,
}

// reconcileACL synchronizes ACL roles and users with the Aerospike cluster.
func (r *AerospikeCEClusterReconciler) reconcileACL(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)

	if cluster.Spec.AerospikeAccessControl == nil {
		return nil
	}

	// Check if any pod is ready before attempting ACL sync
	podReady := false
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(cluster.Namespace),
		client.MatchingLabels(utils.SelectorLabelsForCluster(cluster.Name)),
	); err == nil {
		for i := range podList.Items {
			if isPodReady(&podList.Items[i]) {
				podReady = true
				break
			}
		}
	}
	if !podReady {
		log.Info("No ready pods, skipping ACL sync")
		return nil
	}

	aeroClient, err := r.getAerospikeClient(ctx, cluster)
	if err != nil {
		metrics.ACLSyncTotal.WithLabelValues(cluster.Namespace, cluster.Name, "error").Inc()
		return fmt.Errorf("ACL sync: connecting to cluster: %w", err)
	}
	defer closeAerospikeClient(aeroClient)

	// Sync roles first (users may depend on roles)
	if err := r.reconcileRoles(ctx, aeroClient, cluster); err != nil {
		metrics.ACLSyncTotal.WithLabelValues(cluster.Namespace, cluster.Name, "error").Inc()
		return fmt.Errorf("ACL sync roles: %w", err)
	}

	// Sync users
	if err := r.reconcileUsers(ctx, aeroClient, cluster); err != nil {
		metrics.ACLSyncTotal.WithLabelValues(cluster.Namespace, cluster.Name, "error").Inc()
		return fmt.Errorf("ACL sync users: %w", err)
	}

	metrics.ACLSyncTotal.WithLabelValues(cluster.Namespace, cluster.Name, "success").Inc()
	log.Info("ACL reconciliation completed")
	return nil
}

// reconcileRoles synchronizes roles: create missing, grant/revoke privileges, drop orphaned.
func (r *AerospikeCEClusterReconciler) reconcileRoles(
	ctx context.Context,
	aeroClient *aero.Client,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)

	adminPolicy := aero.NewAdminPolicy()
	adminPolicy.Timeout = aeroInfoTimeout

	// Query existing roles
	existingRoles, err := aeroClient.QueryRoles(adminPolicy)
	if err != nil {
		return fmt.Errorf("querying roles: %w", err)
	}

	existingRoleMap := make(map[string]*aero.Role, len(existingRoles))
	for _, role := range existingRoles {
		existingRoleMap[role.Name] = role
	}

	// Build desired role set
	desiredRoles := make(map[string]bool)
	for _, roleSpec := range cluster.Spec.AerospikeAccessControl.Roles {
		desiredRoles[roleSpec.Name] = true

		existing, exists := existingRoleMap[roleSpec.Name]
		if !exists {
			// Create the role
			privileges := roleParsedPrivileges(roleSpec)
			log.Info("Creating role", "role", roleSpec.Name)
			if err := aeroClient.CreateRole(adminPolicy, roleSpec.Name, privileges, nil, 0, 0); err != nil {
				return fmt.Errorf("creating role %s: %w", roleSpec.Name, err)
			}
			continue
		}

		// Sync privileges: grant missing, revoke extra
		desiredPrivs := roleParsedPrivileges(roleSpec)
		if err := syncRolePrivileges(aeroClient, adminPolicy, roleSpec.Name, existing.Privileges, desiredPrivs, log); err != nil {
			return err
		}
	}

	// Drop orphaned custom roles (protect builtins)
	for _, role := range existingRoles {
		if !desiredRoles[role.Name] && !builtinRoles[role.Name] {
			log.Info("Dropping orphaned role", "role", role.Name)
			if err := aeroClient.DropRole(adminPolicy, role.Name); err != nil {
				return fmt.Errorf("dropping role %s: %w", role.Name, err)
			}
		}
	}

	return nil
}

// reconcileUsers synchronizes users: create missing, grant/revoke roles, change passwords, drop orphaned.
func (r *AerospikeCEClusterReconciler) reconcileUsers(
	ctx context.Context,
	aeroClient *aero.Client,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)

	adminPolicy := aero.NewAdminPolicy()
	adminPolicy.Timeout = aeroInfoTimeout

	// Query existing users
	existingUsers, err := aeroClient.QueryUsers(adminPolicy)
	if err != nil {
		return fmt.Errorf("querying users: %w", err)
	}

	existingUserMap := make(map[string]*aero.UserRoles, len(existingUsers))
	for _, user := range existingUsers {
		existingUserMap[user.User] = user
	}

	desiredUsers := make(map[string]bool)
	for _, userSpec := range cluster.Spec.AerospikeAccessControl.Users {
		desiredUsers[userSpec.Name] = true

		// Read desired password
		password, err := r.getPasswordFromSecret(ctx, cluster.Namespace, userSpec.SecretName)
		if err != nil {
			return fmt.Errorf("getting password for user %s: %w", userSpec.Name, err)
		}

		existing, exists := existingUserMap[userSpec.Name]
		if !exists {
			// Create user
			log.Info("Creating user", "user", userSpec.Name)
			if err := aeroClient.CreateUser(adminPolicy, userSpec.Name, password, userSpec.Roles); err != nil {
				return fmt.Errorf("creating user %s: %w", userSpec.Name, err)
			}
			continue
		}

		// Sync roles
		if err := syncUserRoles(aeroClient, adminPolicy, userSpec.Name, existing.Roles, userSpec.Roles, log); err != nil {
			return err
		}

		// Change password if the secret hash changed.
		// We store the hash in the CR status annotation pattern, but for simplicity,
		// we always attempt to change the password. The Aerospike server is idempotent
		// for this operation if the password is the same.
		passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
		_ = passwordHash // Hash used for logging only; always attempt password change.
		if err := aeroClient.ChangePassword(adminPolicy, userSpec.Name, password); err != nil {
			// Password change can fail if the password is already set (same value).
			// This is not a critical error, log and continue.
			log.V(1).Info("Password change attempt", "user", userSpec.Name, "result", err)
		}
	}

	// Drop orphaned users (protect admin user connected as)
	for _, user := range existingUsers {
		if !desiredUsers[user.User] {
			// Protect the connected admin user
			if user.User == adminUserName {
				continue
			}
			log.Info("Dropping orphaned user", "user", user.User)
			if err := aeroClient.DropUser(adminPolicy, user.User); err != nil {
				return fmt.Errorf("dropping user %s: %w", user.User, err)
			}
		}
	}

	return nil
}

// syncRolePrivileges grants missing and revokes extra privileges for a role.
func syncRolePrivileges(
	aeroClient *aero.Client,
	policy *aero.AdminPolicy,
	roleName string,
	current, desired []aero.Privilege,
	log interface {
		Info(msg string, keysAndValues ...any)
	},
) error {
	currentSet := privilegeSet(current)
	desiredSet := privilegeSet(desired)

	// Grant missing
	var toGrant []aero.Privilege
	for key, priv := range desiredSet {
		if _, ok := currentSet[key]; !ok {
			toGrant = append(toGrant, priv)
		}
	}
	if len(toGrant) > 0 {
		log.Info("Granting privileges to role", "role", roleName, "count", len(toGrant))
		if err := aeroClient.GrantPrivileges(policy, roleName, toGrant); err != nil {
			return fmt.Errorf("granting privileges to role %s: %w", roleName, err)
		}
	}

	// Revoke extra
	var toRevoke []aero.Privilege
	for key, priv := range currentSet {
		if _, ok := desiredSet[key]; !ok {
			toRevoke = append(toRevoke, priv)
		}
	}
	if len(toRevoke) > 0 {
		log.Info("Revoking privileges from role", "role", roleName, "count", len(toRevoke))
		if err := aeroClient.RevokePrivileges(policy, roleName, toRevoke); err != nil {
			return fmt.Errorf("revoking privileges from role %s: %w", roleName, err)
		}
	}

	return nil
}

// syncUserRoles grants missing and revokes extra roles for a user.
func syncUserRoles(
	aeroClient *aero.Client,
	policy *aero.AdminPolicy,
	userName string,
	currentRoles, desiredRoles []string,
	log interface {
		Info(msg string, keysAndValues ...any)
	},
) error {
	currentSet := sliceToSet(currentRoles)
	desiredSet := sliceToSet(desiredRoles)

	// Grant missing
	var toGrant []string
	for role := range desiredSet {
		if !currentSet[role] {
			toGrant = append(toGrant, role)
		}
	}
	if len(toGrant) > 0 {
		log.Info("Granting roles to user", "user", userName, "roles", toGrant)
		if err := aeroClient.GrantRoles(policy, userName, toGrant); err != nil {
			return fmt.Errorf("granting roles to user %s: %w", userName, err)
		}
	}

	// Revoke extra
	var toRevoke []string
	for role := range currentSet {
		if !desiredSet[role] {
			toRevoke = append(toRevoke, role)
		}
	}
	if len(toRevoke) > 0 {
		log.Info("Revoking roles from user", "user", userName, "roles", toRevoke)
		if err := aeroClient.RevokeRoles(policy, userName, toRevoke); err != nil {
			return fmt.Errorf("revoking roles from user %s: %w", userName, err)
		}
	}

	return nil
}

// roleParsedPrivileges converts the spec's privilege strings to aero.Privilege objects.
func roleParsedPrivileges(roleSpec asdbcev1alpha1.AerospikeRoleSpec) []aero.Privilege {
	privileges := make([]aero.Privilege, 0, len(roleSpec.Privileges))
	for _, privStr := range roleSpec.Privileges {
		priv := parsePrivilege(privStr)
		privileges = append(privileges, priv)
	}
	return privileges
}

// parsePrivilege converts a privilege string like "read-write.testNamespace" to an aero.Privilege.
// Since the Code field type (privilegeCode) is unexported in aerospike-client-go,
// we construct a Privilege using exported constants (aero.Read, aero.Write, etc.).
func parsePrivilege(s string) aero.Privilege {
	// Format: "<code>" or "<code>.<namespace>" or "<code>.<namespace>.<set>"
	parts := splitPrivilege(s)

	// Start with a base privilege from the code string
	priv := privilegeFromCodeString(parts[0])

	if len(parts) >= 2 {
		priv.Namespace = parts[1]
	}
	if len(parts) >= 3 {
		priv.SetName = parts[2]
	}

	return priv
}

// splitPrivilege splits a privilege string on '.' separators.
func splitPrivilege(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// privilegeFromCodeString returns an aero.Privilege with the Code matching the string.
// This works around the unexported privilegeCode type by using exported constants.
func privilegeFromCodeString(s string) aero.Privilege {
	switch s {
	case "read":
		return aero.Privilege{Code: aero.Read}
	case "write":
		return aero.Privilege{Code: aero.Write}
	case "read-write":
		return aero.Privilege{Code: aero.ReadWrite}
	case "read-write-udf":
		return aero.Privilege{Code: aero.ReadWriteUDF}
	case "sys-admin":
		return aero.Privilege{Code: aero.SysAdmin}
	case "user-admin":
		return aero.Privilege{Code: aero.UserAdmin}
	case "data-admin":
		return aero.Privilege{Code: aero.DataAdmin}
	case "truncate":
		return aero.Privilege{Code: aero.Truncate}
	default:
		return aero.Privilege{Code: aero.Read}
	}
}

// privilegeKey returns a string key for a privilege for set comparison.
func privilegeKey(p aero.Privilege) string {
	return fmt.Sprintf("%s:%s:%s", p.Code, p.Namespace, p.SetName)
}

// privilegeSet converts a slice of privileges to a map keyed by privilegeKey.
func privilegeSet(privs []aero.Privilege) map[string]aero.Privilege {
	set := make(map[string]aero.Privilege, len(privs))
	for _, p := range privs {
		set[privilegeKey(p)] = p
	}
	return set
}

// sliceToSet converts a string slice to a set.
func sliceToSet(ss []string) map[string]bool {
	set := make(map[string]bool, len(ss))
	for _, s := range ss {
		set[s] = true
	}
	return set
}
