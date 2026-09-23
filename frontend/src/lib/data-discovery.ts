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
      { key: "sharedDrive", labelKey: "sharedDrive", kind: "text", required: true, placeholder: "Finance" },
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

export function decodeTarget(source: SourceType, encoded: string): Record<string, string> {
  const fields = targetFields(source);
  const parts = encoded.split("/").filter(Boolean);
  const values = emptyTargetValues(source);

  fields.forEach((field, index) => {
    if (index === fields.length - 1) {
      values[field.key] = parts.slice(index).join("/");
      return;
    }

    values[field.key] = parts[index] ?? "";
  });

  return values;
}
