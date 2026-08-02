# Phase 0 — Create the organization

**No Terraform in this phase.** It's console work, domain-admin work, and a handful of `gcloud`
commands. It's also the least reversible thing in the tutorial, so read it through once before you
start clicking.

## 1. Why this comes first

Your account has 0 organizations and 8 projects, all owned directly by a personal Gmail. That
arrangement can't support what you're building:

- **Project quota.** Personal accounts get a small project quota that grows slowly with billing
  history. At 8 projects you're near the ceiling, and deleted projects hold their ID for 30 days.
  A project-per-ephemeral-environment factory needs quota you can actually request — which is an
  org-level conversation.
- **No Google Groups.** Without a Cloud Identity domain, "stakeholders can read staging" means
  individual email addresses in IAM bindings, and every person who joins or leaves is a
  `terraform apply`. With groups, Terraform binds a role to `gcp-staging-viewers@yourdomain.com`
  once and membership changes outside Terraform forever.
- **No folders.** Folders are how "these principals reach this tier and nothing else" gets
  expressed structurally instead of by repeating bindings in every project.
- **No org policies.** Constraints enforced from above rather than hoped for.

Cloud Identity Free costs nothing. The only thing with a price tag is the domain, which you already
own — `infra/modules/cloud-run-domain-mapping` and staging's `api_domain` variable both depend on it.

The real cost is effort: you'll create a new admin identity on your domain, and your existing
projects have to be moved under the org. Doing that now, with 8 projects, is much easier than doing
it later with a project factory and folder-inherited IAM built on top.

## 2. Clean up first

Migration is per-project, and dead projects cost quota. Before anything else:

```bash
gcloud projects list --format="table(projectId, name, lifecycleState, createTime)"
```

For each one you don't need:

```bash
gcloud projects delete PROJECT_ID
```

> **The ID is held for 30 days.** A deleted project sits in `DELETE_REQUESTED` and its ID stays
> reserved. You can't reuse the name, and it may still count against quota during that window. So
> delete now, not the day you need the slot.

Note which projects are `fang-gcp` (prod) and `fang-gcp-staging` — those two matter for Phase 2.

## 3. Sign up for Cloud Identity Free

Go to <https://workspace.google.com/gcpidentity/signup> and choose the **free** edition. Watch for
this — the signup flow steers toward Workspace, which is paid. You want Cloud Identity Free.

You'll provide:

- your domain name
- a new admin username on that domain, e.g. `nick@yourdomain.com`

**This is a new identity, separate from your Gmail.** Your Gmail owns the projects; this new user
will own the organization. Getting them connected is section 5, and it's the step people get stuck
on.

Verify the domain by adding the TXT record Google gives you to your DNS. Propagation is usually
minutes. Confirm with:

```bash
dig +short TXT yourdomain.com
```

## 4. The organization appears on its own

You don't create the organization explicitly. Once the domain is verified, GCP creates the org
resource the first time a user from that domain signs in to the Cloud Console.

Sign in as `nick@yourdomain.com`, open the Cloud Console, then:

```bash
gcloud auth login nick@yourdomain.com
gcloud organizations list
```

```
DISPLAY_NAME      ID              DIRECTORY_CUSTOMER_ID
yourdomain.com    123456789012    C01abcdef
```

Save that ID — nearly every command from here on needs it.

```bash
export ORG_ID=123456789012
```

## 5. Connect your Gmail to the new organization

This is the fiddly part. Your Gmail account owns the 8 projects. Your new domain user owns the org.
Neither can complete the migration alone: moving a project needs permission on **both** the project
and the destination org.

Two roles are involved, and it's worth knowing which does what:

| Role | Granted on | Lets you |
|---|---|---|
| `roles/resourcemanager.projectCreator` | the org | place a project under it |
| `roles/resourcemanager.projectMover` | the project (or its parent) | move a project out of where it is |

> Looking for these in the console role picker? They appear as **Project Creator** and **Project
> Mover** (filter by "Resource Manager"), on the **organization's** IAM page. Don't reach for
> Owner/Editor — those are far broader roles that happen to contain similar words.

Your Gmail already effectively has the second, as project owner. It needs the first.

### First, unblock the org — it was born restricted

Organizations created on or after **May 3, 2024** are "secure-by-default": Google enforces a
bundle of org policies at creation, and one of them is `constraints/iam.allowedPolicyMemberDomains`
(domain-restricted sharing), pre-restricted to your Cloud Identity domain. Until you deal with it,
every binding below fails with:

```
IAM policy update failed
The 'Domain Restricted Sharing' organization policy (constraints/iam.allowedPolicyMemberDomains)
is enforced. Only principals in allowed domains can be added as principals in the policy.
```

You can't allow-list the Gmail, either: the constraint's allowed values are Google Workspace /
Cloud Identity **customer IDs** and **organization principal sets** — managed identity pools. A
consumer Gmail account belongs to neither. That's by design; excluding unmanaged identities is
the constraint's entire job.

The fix is **delete → bind → restore**. Enforcement is not retroactive — bindings that exist when
the policy comes back remain valid — so the org runs unrestricted for minutes, not until Phase 1.

As `nick@yourdomain.com`:

```bash
# Organization Administrator (auto-granted to the super admin who created the org) can set
# org IAM, but org *policies* need a separate role. Grant it to yourself:
gcloud organizations add-iam-policy-binding "$ORG_ID" \
  --member="user:nick@yourdomain.com" \
  --role="roles/orgpolicy.policyAdmin"

# Back up Google's policy exactly as created, then delete it:
gcloud org-policies describe iam.allowedPolicyMemberDomains \
  --organization="$ORG_ID" > drs-policy-backup.yaml
gcloud org-policies delete iam.allowedPolicyMemberDomains --organization="$ORG_ID"
```

> If the self-grant fails with a permission error, your domain user isn't a Cloud Identity super
> admin yet (the auto-granted Organization Administrator role goes to the super admin who created
> the org). Fix it at <https://admin.google.com> under **Account → Admin roles → Super Admin**,
> then retry. Super admin is a Cloud Identity role; Organization Administrator is a GCP IAM role.
> They're different things and having one doesn't give you the other.

Give the deletion a minute or two to propagate, then make the bindings:

```bash
gcloud organizations add-iam-policy-binding "$ORG_ID" \
  --member="user:fang.nicholas@gmail.com" \
  --role="roles/resourcemanager.projectCreator"

gcloud organizations add-iam-policy-binding "$ORG_ID" \
  --member="user:fang.nicholas@gmail.com" \
  --role="roles/resourcemanager.projectMover"
```

While you're here, give your domain user the org-admin role so it can manage IAM and policies,
and the folder-creator role for §6 — Organization Administrator manages IAM but does **not**
include folder creation, which needs its own role at the org level:

```bash
gcloud organizations add-iam-policy-binding "$ORG_ID" \
  --member="user:nick@yourdomain.com" \
  --role="roles/resourcemanager.organizationAdmin"

gcloud organizations add-iam-policy-binding "$ORG_ID" \
  --member="user:nick@yourdomain.com" \
  --role="roles/resourcemanager.folderCreator"
```

Then restore the domain restriction **in the same sitting**:

```bash
gcloud org-policies set-policy drs-policy-backup.yaml

# Confirm it's back, and the Gmail's bindings survived:
gcloud org-policies describe iam.allowedPolicyMemberDomains --organization="$ORG_ID"
gcloud organizations get-iam-policy "$ORG_ID" \
  --flatten="bindings[].members" --filter="bindings.members:fang.nicholas" \
  --format="table(bindings.role)"
```

> If `set-policy` rejects the backup file, strip any `etag:` and `updateTime:` lines and retry.
> Leave the other secure-by-default policies (`iam.disableServiceAccountKeyCreation` and friends)
> alone — deploys authenticate with Workload Identity Federation, so nothing needs service account
> keys, and Phase 1 imports these policies into Terraform.

## 6. Create the folders

Still as your domain user — the Gmail has no business here. If these return `PERMISSION_DENIED`,
you're missing the `folderCreator` grant from the end of §5.

```bash
for f in platform dev staging prod; do
  gcloud resource-manager folders create \
    --display-name="$f" \
    --organization="$ORG_ID"
done

gcloud resource-manager folders list --organization="$ORG_ID"
```

Record the four folder IDs — Phase 1 needs them:

```bash
export FOLDER_PLATFORM=111111111111
export FOLDER_DEV=222222222222
export FOLDER_STAGING=333333333333
export FOLDER_PROD=444444444444
```

You're creating these by hand rather than in Terraform for the same reason the state bucket is a
chicken-and-egg problem: Terraform needs somewhere to put state, and that somewhere lives in the
platform project, which lives in a folder. Phase 1 imports these into Terraform once there's a
backend to hold them.

## 7. Move the existing projects in

Switch back to the account that owns the projects:

```bash
gcloud auth login fang.nicholas@gmail.com

gcloud beta projects move fang-gcp-staging \
  --folder="$FOLDER_STAGING"

gcloud beta projects move fang-gcp \
  --folder="$FOLDER_PROD"
```

Move anything else you kept into whichever folder fits. If nothing fits, leave it at the org root
for now — you can move it later, and an unsorted project is better than a wrong folder.

Confirm:

```bash
gcloud projects list --format="table(projectId, parent.type, parent.id)"
```

### The gotcha: projects inherit policy on arrival

The moment a project lands in a folder, it inherits everything above it. That's the point, and it's
also how people break production during a migration.

Domain-restricted sharing (`constraints/iam.allowedPolicyMemberDomains`) is enforced again — §5
restored it after the Gmail bindings. Your dashboard API has a binding this constraint forbids:

```hcl
# infra/modules/cloud-run-aggregator/main.tf:55-60
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  role   = "roles/run.invoker"
  member = "allUsers"        # <-- a domain-restriction policy blocks this
}
```

Two facts about how the constraint behaves keep the move safe, and both matter:

- **Enforcement is not retroactive.** The existing `allUsers` binding survives the project move
  untouched — moving a project changes its parent, not its IAM policy. The public API keeps
  answering.
- **New policy writes are what's blocked.** The next `terraform apply` that needs to create or
  recreate that binding fails until Phase 1 lands the folder exemptions.

So between now and Phase 1: move projects freely, but don't change staging or prod infrastructure
that touches the public invoker, and don't run an apply that would recreate it.

## 8. Create the groups

At <https://admin.google.com> → **Directory → Groups**, create four:

| Group | Purpose | Gets, eventually |
|---|---|---|
| `gcp-platform-admins@yourdomain.com` | you, and anyone who can change the platform | org-level admin |
| `gcp-developers@yourdomain.com` | engineers who spin up dev environments | project creation in `folders/dev` |
| `gcp-staging-viewers@yourdomain.com` | stakeholders who need to see staging | read-only on `folders/staging` |
| `gcp-prod-oncall@yourdomain.com` | whoever can touch prod | scoped access to `folders/prod` |

The create-group wizard's second page ("Group settings") defaults to mailing-list behavior. These
groups grant cloud permissions to their members, so the deciding rule is: **membership must never
be self-service** — joining `gcp-platform-admins` would mean granting yourself org-level admin.
For all four groups:

- **Access type: Restricted** — not the default "Public", which lets anyone in the org join
  themselves.
- **Who can join the group: Only invited users** — the setting that actually matters; membership
  changes stay a deliberate admin action.
- **Allow external members: both boxes unchecked** — domain-restricted sharing validates the
  *group's* domain in IAM bindings, not its members', so an external member would ride the
  group's bindings past that control.
- The access-settings grid (who can contact owners / view conversations / post) governs the group
  as a mailing list — irrelevant here; the Restricted presets are fine.

Add `nick@yourdomain.com` to `gcp-platform-admins` and leave the rest empty. They exist so Phase 3
can bind roles to them — an empty group with a role attached is fine and is exactly how you want
this to work. Adding a stakeholder later becomes a click in the admin console, not a Terraform run.

Don't add your Gmail to these. It keeps its project ownership for now and drops out of the picture
once Phase 2 is verified.

## 9. Break-glass — do this before you depend on the org

You've just made a domain and a single admin identity the root of access to everything. Before the
org is load-bearing, make sure losing either one isn't terminal.

Three things, none of which take long:

**A second super admin.** Create `nick-breakglass@yourdomain.com` in the admin console, make it a
super admin, give it a long unique password stored in a password manager, and enrol a separate
second factor — ideally a hardware key kept somewhere physical. Don't use it day to day. A single
super admin means one lost phone or one lockout costs you administrative access to eight projects.

**Domain auto-renew.** The entire org hangs off DNS ownership. If the registration lapses, domain
verification fails and recovery becomes a support conversation. Turn on auto-renew and confirm the
billing card on the registrar isn't the one that expires next year.

**Recovery codes stored outside GCP.** Generate super-admin recovery codes and keep them somewhere
that doesn't require a working Google account to reach — a password manager, or printed. Storing
them in Drive is a circular dependency you'll discover at the worst moment.

Also worth knowing: your personal Gmail keeps `projectCreator` and `projectMover` on the org from
§5, and remains an owner on the projects it created. That's a second, independent path in. Don't
remove it until you've verified the break-glass account works.

## 10. Request project quota

The ephemeral environment factory in Phase 4 creates a project per environment. Default quota won't
support that.

Console → **IAM & Admin → Quotas & System Limits**, filter for the Cloud Resource Manager project
creation limit, and request an increase. Ask for something defensible — 50 is plenty for per-branch
environments with a reaper cleaning up.

Approval takes a few days, so file it now and continue. Phases 1 through 3 don't depend on it.

## 11. Move billing under the org (optional, do it now if at all)

Your billing account is still owned by your Gmail. It works as-is — the moved projects keep
billing normally. But an org-owned billing account means budget alerts, billing exports, and
`roles/billing.user` can be managed as part of the same IAM story rather than being personal
settings on your account.

Console → **Billing → Account management → Change organization**. If you're going to do it, doing
it now avoids a second migration later.

---

## Verification gate

All four must pass before Phase 1.

```bash
# 1. The organization exists
gcloud organizations list
```
Shows your domain and an org ID.

```bash
# 2. Four folders exist under it
gcloud resource-manager folders list --organization="$ORG_ID"
```
Shows `platform`, `dev`, `staging`, `prod`.

```bash
# 3. Both real projects sit in the right folders
gcloud projects list --format="table(projectId, parent.type, parent.id)"
```
`fang-gcp-staging` → `folder` / `$FOLDER_STAGING`; `fang-gcp` → `folder` / `$FOLDER_PROD`.

```bash
# 4. Nothing broke — the public API still answers
curl -sS -o /dev/null -w '%{http_code}\n' https://<your-staging-api-domain>/api/v1/dashboard
```
Expect `200`. If you get `403`, an inherited policy is blocking `allUsers` — see section 7.

```bash
# 5. Domain restriction is back on, and the Gmail kept its org roles
gcloud org-policies describe iam.allowedPolicyMemberDomains --organization="$ORG_ID"
gcloud organizations get-iam-policy "$ORG_ID" \
  --flatten="bindings[].members" --filter="bindings.members:fang.nicholas" \
  --format="table(bindings.role)"
```
The policy exists, and the roles list includes `projectCreator` and `projectMover`. If you want
proof the restriction is really enforcing, try adding any Gmail binding — it should fail with the
same Domain Restricted Sharing error §5 started with.

Also worth confirming your existing deploys still work, since the projects moved underneath them:
re-run one GitHub Actions deploy workflow and check it completes. Workload Identity Federation
bindings are project-scoped and survive a folder move, but it's a cheap thing to verify while the
change is fresh.

---

**Next:** [01-bootstrap-and-platform.md](01-bootstrap-and-platform.md) — the first Terraform you
write, and the state bucket everything else depends on.
