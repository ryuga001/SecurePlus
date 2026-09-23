import {
    AlertCircle,
    Building2,
    ChartNoAxesCombined,
    FileText,
    LayoutDashboard,
    Mail,
    Palette,
    PlugZap,
    Radar,
    Regex,
    ScanSearch,
    ScrollText,
    Send,
    Settings,
    ShieldAlert,
    ShieldCheck,
    SlidersHorizontal,
    UserRound,
    Users,
    UsersRound,
    type LucideIcon,
} from "lucide-react";

export type RouteNode = {
    labelKey: string;
    routePath?: string;
    icon?: LucideIcon;
    privileges: readonly string[];
    hidden?: boolean;
    defaultOpen?: boolean;
    children?: Record<string, RouteNode>;
};

export type RouteMap = Record<string, RouteNode>;

export const routes = {
    dashboard: {
        labelKey: "dashboard",
        routePath: "/dashboard",
        icon: LayoutDashboard,
        privileges: [],
    },

    emailProtection: {
        labelKey: "emailProtection",
        icon: Mail,
        privileges: [],
        defaultOpen: true,
        children: {
            configurations: {
                labelKey: "configurations",
                routePath: "/admin/email/configurations",
                icon: SlidersHorizontal,
                privileges: [],
            },
            policies: {
                labelKey: "policies",
                routePath: "/admin/policies",
                icon: ShieldCheck,
                privileges: [],
            },
            rules: {
                labelKey: "rules",
                routePath: "/admin/rules",
                icon: Regex,
                privileges: [],
            },
            users: {
                labelKey: "users",
                routePath: "/admin/email/users",
                icon: Users,
                privileges: [],
            },
            groups: {
                labelKey: "groups",
                routePath: "/admin/email/groups",
                icon: UsersRound,
                privileges: [],
            },
            audits: {
                labelKey: "audits",
                routePath: "/admin/email/audits",
                icon: FileText,
                privileges: ["admin.email.audit.view"],
                children: {
                    incidents: {
                        labelKey: "incidents",
                        routePath: "/admin/email/audits/incidents",
                        icon: ShieldAlert,
                        privileges: ["admin.email.incident.view"],
                    },
                    delivery: {
                        labelKey: "delivery",
                        routePath: "/admin/email/audits/delivery",
                        icon: Send,
                        privileges: ["admin.email.audit.view"],
                    },
                },
            },
            alerts: {
                labelKey: "alert",
                routePath: "/admin/email/alert",
                icon: AlertCircle,
                privileges: ["admin.email.alert.view"],
            }
        },
    },
    dataDiscovery: {
        labelKey: "dataDiscovery",
        routePath: "/admin/data-discovery",
        icon: Radar,
        privileges: ["admin.discovery.configuration.view"],
        children: {
            dd_configurations: {
                labelKey: "discoverySources",
                routePath: "/admin/data-discovery/configurations",
                icon: PlugZap,
                privileges: ["admin.discovery.configuration.view"],
            },
            dd_policies: {
                labelKey: "discoveryPolicies",
                routePath: "/admin/data-discovery/policies",
                icon: ShieldCheck,
                privileges: ["admin.discovery.policy.view"],
            },
            dd_scans: {
                labelKey: "discoveryScans",
                routePath: "/admin/data-discovery/scans",
                icon: ScanSearch,
                privileges: ["admin.discovery.policy.view"],
            },
            dd_analysis: {
                labelKey: "discoveryAnalysis",
                routePath: "/admin/data-discovery/analysis",
                icon: ChartNoAxesCombined,
                privileges: ["admin.discovery.policy.view"],
            },
        },
    },
    administration: {
        labelKey: "administration",
        icon: Building2,
        privileges: [],
        defaultOpen: true,
        children: {
            admin_console: {
                labelKey: "adminConsole",
                routePath: "/admin/console",
                icon: ScrollText,
                privileges: [],
                children: {
                    user_roles: {
                        labelKey: "userRoles",
                        routePath: "/admin/console/user-roles",
                        icon: Users,
                        privileges: [],
                    },
                    user_management: {
                        labelKey: "userManagement",
                        routePath: "/admin/console/user-management",
                        icon: Users,
                        privileges: [],
                    },
                }
            },
            settings: {
                labelKey: "settings",
                routePath: "/admin/settings",
                icon: Settings,
                privileges: [],
                children: {
                    profile: {
                        labelKey: "profile",
                        routePath: "/admin/settings/profile",
                        icon: UserRound,
                        privileges: [],
                    },
                    branding: {
                        labelKey: "branding",
                        routePath: "/admin/settings/branding",
                        icon: Palette,
                        privileges: ["admin.branding.edit"],
                    },
                }
            },
        },
    },
} as const satisfies RouteMap;