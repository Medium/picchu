It's not good to assume that resources exist. It's possible for a particular
cluster to be offline for a period of time and miss certain updates. Try to
use controllerutil.CreateOrUpdate for all resources you are changing and to
have them completely filled out. Simple edits are not supported.
