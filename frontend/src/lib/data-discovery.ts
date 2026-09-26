import {
  Boxes,
  Building2,
  Cloud,
  FolderOpen,
  HardDrive,
  KeyRound,
  Server,
  type LucideIcon,
} from "lucide-react";

export const CONFIGURATION_TYPES = [
  "MICROSOFT_ENTRA_ACCOUNT",
  "AZURE_STORAGE_ACCOUNT",
  "GOOGLE_SERVICE_ACCOUNT",
  "AWS_IAM",
] as const;

export type ConfigurationType = (typeof CONFIGURATION_TYPES)[number];

export const SOURCE_TYPES = [
  "SHARE_POINT",
  "ONE_DRIVE",
  "AZURE_BLOB",
  "GOOGLE_DRIVE",
  "AWS_S3",
] as const;

export type SourceType = (typeof SOURCE_TYPES)[number];

export const DISCOVERY_STATUSES = ["ACTIVE", "INACTIVE"] as const;

export type DiscoveryStatus = (typeof DISCOVERY_STATUSES)[number];

export const SCAN_STATUSES = ["PENDING", "RUNNING", "PARTIAL", "COMPLETED", "FAILED"] as const;

export type ScanStatus = (typeof SCAN_STATUSES)[number];

export const FILE_RESULT_STATUSES = ["SUCCEEDED", "FAILED"] as const;

export type FileResultStatus = (typeof FILE_RESULT_STATUSES)[number];

export const SCAN_POLL_INTERVAL = 5000;

export function isScanActive(status: ScanStatus | string | undefined) {
  return status === "PENDING" || status === "RUNNING";
}

export function formatBytes(size: number) {
  if (!Number.isFinite(size) || size < 0) return "—";
  if (size < 1024) return `${size} B`;

  const units = ["KB", "MB", "GB", "TB"];
  let value = size / 1024;
  let unit = 0;

  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }

  return `${value.toFixed(value < 10 ? 2 : 1)} ${units[unit]}`;
}

export function formatDuration(ms: number) {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)} ms`;

  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)} s`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${Math.round(seconds % 60)}s`;

  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

export type FieldKind = "text" | "secret" | "gcpServiceAccountKey";

export type FieldDescriptor = {
  key: string;
  labelKey: string;
  hintKey?: string;
  kind: FieldKind;
  required: boolean;
  placeholder?: string;
  pattern?: RegExp;
  mono?: boolean;
  derived?: readonly string[];
};

const UUID_OR_DOMAIN =
  /^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[a-z0-9-]+(\.[a-z0-9-]+)+)$/i;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const TENANT_ID: FieldDescriptor = {
  key: "tenantId",
  labelKey: "tenantId",
  hintKey: "tenantId",
  kind: "text",
  required: true,
  mono: true,
  pattern: UUID_OR_DOMAIN,
  placeholder: "00000000-0000-0000-0000-000000000000",
};

const CLIENT_ID: FieldDescriptor = {
  key: "clientId",
  labelKey: "clientId",
  kind: "text",
  required: true,
  mono: true,
  pattern: UUID,
  placeholder: "00000000-0000-0000-0000-000000000000",
};

const CLIENT_SECRET: FieldDescriptor = {
  key: "clientSecret",
  labelKey: "clientSecret",
  kind: "secret",
  required: true,
};

export type ConfigurationTypeDescriptor = {
  labelKey: string;
  hintKey: string;
  icon: LucideIcon;
  monogram: string;
  fields: readonly FieldDescriptor[];
};

export const CONFIGURATION_TYPE_META = {
  MICROSOFT_ENTRA_ACCOUNT: {
    labelKey: "MICROSOFT_ENTRA_ACCOUNT",
    hintKey: "MICROSOFT_ENTRA_ACCOUNT",
    icon: Building2,
    monogram: "EN",
    fields: [TENANT_ID, CLIENT_ID, CLIENT_SECRET],
  },
  AZURE_STORAGE_ACCOUNT: {
    labelKey: "AZURE_STORAGE_ACCOUNT",
    hintKey: "AZURE_STORAGE_ACCOUNT",
    icon: Server,
    monogram: "AZ",
    fields: [
      {
        key: "storageAccount",
        labelKey: "storageAccount",
        hintKey: "storageAccount",
        kind: "text",
        required: true,
        mono: true,
        pattern: /^[a-z0-9]{3,24}$/,
        placeholder: "mystorageaccount",
      },
      TENANT_ID,
      CLIENT_ID,
      CLIENT_SECRET,
    ],
  },
  GOOGLE_SERVICE_ACCOUNT: {
    labelKey: "GOOGLE_SERVICE_ACCOUNT",
    hintKey: "GOOGLE_SERVICE_ACCOUNT",
    icon: Cloud,
    monogram: "GC",
    fields: [
      {
        key: "serviceAccountKey",
        labelKey: "serviceAccountKey",
        hintKey: "serviceAccountKey",
        kind: "gcpServiceAccountKey",
        required: true,
        derived: ["projectId", "clientEmail"],
      },
      {
        key: "subject",
        labelKey: "subject",
        hintKey: "subject",
        kind: "text",
        required: false,
        pattern: /^[^@\s]+@[^@\s]+\.[^@\s]+$/,
        placeholder: "admin@example.com",
      },
    ],
  },
  AWS_IAM: {
    labelKey: "AWS_IAM",
    hintKey: "AWS_IAM",
    icon: KeyRound,
    monogram: "AW",
    fields: [
      {
        key: "accessKeyId",
        labelKey: "accessKeyId",
        kind: "text",
        required: true,
        mono: true,
        pattern: /^[A-Z0-9]{16,128}$/,
        placeholder: "AKIA…",
      },
      {
        key: "secretAccessKey",
        labelKey: "secretAccessKey",
        kind: "secret",
        required: true,
      },
      {
        key: "region",
        labelKey: "region",
        hintKey: "region",
        kind: "text",
        required: true,
        mono: true,
        pattern: /^[a-z]{2}(-[a-z]+)+-\d$/,
        placeholder: "ap-south-1",
      },
    ],
  },
} as const satisfies Record<ConfigurationType, ConfigurationTypeDescriptor>;

export type SourceTypeDescriptor = {
  labelKey: string;
  icon: LucideIcon;
  configurationTypes: readonly [ConfigurationType, ...ConfigurationType[]];
  targetFields: readonly FieldDescriptor[];
};

export const SOURCE_TYPE_META = {
  SHARE_POINT: {
    labelKey: "SHARE_POINT",
    icon: Boxes,
    configurationTypes: ["MICROSOFT_ENTRA_ACCOUNT"],
    targetFields: [
      { key: "site", labelKey: "site", kind: "text", required: true, mono: true, placeholder: "/sites/finance" },
      { key: "documentLibrary", labelKey: "documentLibrary", kind: "text", required: false, placeholder: "Documents" },
      { key: "folder", labelKey: "folder", kind: "text", required: false, mono: true, placeholder: "/2025/Q1" },
    ],
  },
  ONE_DRIVE: {
    labelKey: "ONE_DRIVE",
    icon: HardDrive,
    configurationTypes: ["MICROSOFT_ENTRA_ACCOUNT"],
    targetFields: [
      { key: "drive", labelKey: "drive", kind: "text", required: true, placeholder: "user@example.com" },
      { key: "folder", labelKey: "folder", kind: "text", required: false, mono: true, placeholder: "/Documents" },
    ],
  },
  AZURE_BLOB: {
    labelKey: "AZURE_BLOB",
    icon: Server,
    configurationTypes: ["AZURE_STORAGE_ACCOUNT"],
    targetFields: [
      {
        key: "container",
        labelKey: "container",
        kind: "text",
        required: true,
        mono: true,
        pattern: /^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$/,
        placeholder: "documents",
      },
      { key: "prefix", labelKey: "prefix", kind: "text", required: false, mono: true, placeholder: "invoices/" },
    ],
  },
  GOOGLE_DRIVE: {
    labelKey: "GOOGLE_DRIVE",
    icon: Cloud,
    configurationTypes: ["GOOGLE_SERVICE_ACCOUNT"],
    targetFields: [
      { key: "sharedDrive", labelKey: "sharedDrive", kind: "text", required: true, placeholder: "Finance or user@example.com" },
      { key: "folder", labelKey: "folder", kind: "text", required: false, mono: true, placeholder: "/Reports" },
    ],
  },
  AWS_S3: {
    labelKey: "AWS_S3",
    icon: FolderOpen,
    configurationTypes: ["AWS_IAM"],
    targetFields: [
      {
        key: "bucket",
        labelKey: "bucket",
        kind: "text",
        required: true,
        mono: true,
        pattern: /^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$/,
        placeholder: "my-bucket",
      },
      { key: "prefix", labelKey: "prefix", kind: "text", required: false, mono: true, placeholder: "raw/2025/" },
    ],
  },
} as const satisfies Record<SourceType, SourceTypeDescriptor>;

export const SOURCE_TYPES_BY_CONFIGURATION_TYPE = CONFIGURATION_TYPES.reduce(
  (map, configurationType) => {
    map[configurationType] = SOURCE_TYPES.filter((sourceType) =>
      (SOURCE_TYPE_META[sourceType].configurationTypes as readonly ConfigurationType[]).includes(
        configurationType,
      ),
    );

    return map;
  },
  {} as Record<ConfigurationType, readonly SourceType[]>,
);

export function supportedSourceTypes(type: ConfigurationType) {
  return SOURCE_TYPES_BY_CONFIGURATION_TYPE[type];
}

export function configurationTypesFor(source: SourceType) {
  return SOURCE_TYPE_META[source].configurationTypes;
}

export function isCompatible(source: SourceType, type: ConfigurationType) {
  return (SOURCE_TYPE_META[source].configurationTypes as readonly ConfigurationType[]).includes(type);
}

export function credentialFields(type: ConfigurationType): readonly FieldDescriptor[] {
  return CONFIGURATION_TYPE_META[type].fields;
}

export function targetFields(source: SourceType): readonly FieldDescriptor[] {
  return SOURCE_TYPE_META[source].targetFields;
}

export function secretFieldKeys(type: ConfigurationType) {
  return credentialFields(type)
    .filter((field) => field.kind === "secret" || field.kind === "gcpServiceAccountKey")
    .map((field) => field.key);
}

export function emptyTargetValues(source: SourceType): Record<string, string> {
  return Object.fromEntries(targetFields(source).map((field) => [field.key, ""]));
}

export function encodeTarget(source: SourceType, values: Record<string, string>) {
  return targetFields(source)
    .map((field) => (values[field.key] ?? "").trim())
    .filter(Boolean)
    .join("/");
}

function decodeSharePointTarget(encoded: string): Record<string, string> {
  let value = encoded.trim();
  let origin = "";

  const absolute = /^(https?:\/\/[^/]+)(.*)$/i.exec(value);
  if (absolute) {
    origin = absolute[1];
    value = absolute[2];
  }

  const parts = value.split("/");
  let index = 0;

  while (index < parts.length && parts[index] === "") index++;

  let site = "/";
  if (index + 1 < parts.length && /^(sites|teams)$/i.test(parts[index]) && parts[index + 1]) {
    site = `/${parts[index]}/${parts[index + 1]}`;
    index += 2;
  }

  let documentLibrary = "";
  if (index < parts.length && parts[index] !== "") {
    documentLibrary = parts[index];
    index++;
  }

  const folder = parts.slice(index).filter(Boolean);

  return {
    site: origin ? origin + (site === "/" ? "" : site) : site,
    documentLibrary,
    folder: folder.length > 0 ? `/${folder.join("/")}` : "",
  };
}

export function decodeTarget(source: SourceType, encoded: string): Record<string, string> {
  if (source === "SHARE_POINT") return decodeSharePointTarget(encoded);

  const fields = targetFields(source);
  const values = emptyTargetValues(source);
  let rest = encoded.trim();

  fields.forEach((field, index) => {
    if (index === fields.length - 1) {
      values[field.key] = rest;
      return;
    }

    rest = rest.replace(/^\/+/, "");

    const slash = rest.indexOf("/");
    if (slash < 0) {
      values[field.key] = rest;
      rest = "";
      return;
    }

    values[field.key] = rest.slice(0, slash);
    rest = rest.slice(slash + 1);
  });

  return values;
}
