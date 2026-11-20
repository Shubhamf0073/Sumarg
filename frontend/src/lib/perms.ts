/** UI hint only: the API enforces permissions. */
export type PermissionSet = ReadonlySet<string>;

export function can(perms: PermissionSet | undefined, permission: string): boolean {
  return perms?.has(permission) ?? false;
}
