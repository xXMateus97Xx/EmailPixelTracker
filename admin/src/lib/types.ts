// Shapes of the Email Pixel Tracker API (see api/docs/openapi.yaml).

export type Role = "SuperAdmin" | "Admin" | "ContentManager" | "API";

export type Page<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};

export type Tenant = {
  id: string;
  name: string;
  active: boolean;
  createdAtUtc: string;
};

export type User = {
  id: string;
  name: string;
  username: string;
  tenantId?: string;
  active: boolean;
  createdAtUtc: string;
  role: Role;
};

export type Permission = { read: boolean; write: boolean };

export type Me = {
  user: User;
  impersonatedTenantId?: string;
  readOnly: boolean;
  tenant?: Tenant;
  permissions: {
    tenants: Permission;
    users: Permission;
    campaigns: Permission;
    impersonate: boolean;
  };
};

export type RoleInfo = { name: Role; assignable: boolean };

export type Token = { token: string; expiresAtUtc: string };

export type RecaptchaConfig = { enabled: boolean; siteKey?: string; action?: string };

export type Campaign = {
  id: string;
  name: string;
  tenantId: string;
  active: boolean;
  createdAtUtc: string;
  expireInUtc: string;
  pixelColorR: number;
  pixelColorG: number;
  pixelColorB: number;
  pixelColorA: number;
};

export type ViewSummary = {
  campaignId: string;
  fromUtc?: string;
  toUtc?: string;
  totalViews: number;
  uniqueViews: number;
};

export type DailyViews = {
  campaignId: string;
  days: { date: string; totalViews: number; uniqueViews: number }[];
};

export type ViewGroup = { name: string; totalViews: number; uniqueViews: number };

export type UserAgentReport = {
  campaignId: string;
  fromUtc?: string;
  toUtc?: string;
  clients: ViewGroup[];
  operatingSystems: ViewGroup[];
  devices: ViewGroup[];
  userAgents: {
    /** Empty for the views without a user agent. */
    userAgent: string;
    client: string;
    os: string;
    device: string;
    totalViews: number;
    uniqueViews: number;
  }[];
};

export type CampaignView = {
  id: string;
  viewedOnUtc: string;
  externalUserId?: string;
  ipAddress: string;
  userAgent?: string;
};

export type FieldError = { field: string; message: string };

export type ErrorBody = {
  error: string;
  fields?: FieldError[];
  requestId?: string;
};
