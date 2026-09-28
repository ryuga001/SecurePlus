import {
    AlertCircle,
    Building2,
    ChartNoAxesCombined,
    FileText,
    KeyRound,
    LayoutDashboard,
    Mail,
    Palette,
    PlugZap,
    Radar,
    Regex,
    ScanSearch,
    ScanText,
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
    adminOnly?: boolean;
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
                privileges: ["admin.email.provider.view"],
            },
            policies: {
                labelKey: "policies",
                routePath: "/admin/policies",
                icon: ShieldCheck,
                privileges: ["admin.policy.view"],
            },
            users: {
                labelKey: "users",
                routePath: "/admin/email/users",
                icon: Users,
                privileges: ["admin.email.user.view"],
            },
            groups: {
                labelKey: "groups",
                routePath: "/admin/email/groups",
                icon: UsersRound,
                privileges: ["admin.email.group.view"],
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
                privileges: ["admin.discovery.scan.view"],
            },
            dd_analysis: {
                labelKey: "discoveryAnalysis",
                routePath: "/admin/data-discovery/analysis",
                icon: ChartNoAxesCombined,
                privileges: ["admin.discovery.policy.view"],
            },
        },
    },
    contentInspection: {
        labelKey: "contentInspection",
        icon: ScanText,
        privileges: [],
        defaultOpen: true,
        children: {
            ci_rules: {
                labelKey: "detectionRules",
                routePath: "/admin/content-inspection/rules",
                icon: Regex,
                privileges: ["admin.rule.view"],
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
                adminOnly: true,
                children: {
                    user_roles: {
                        labelKey: "userRoles",
                        routePath: "/admin/console/user-roles",
                        icon: KeyRound,
                        privileges: [],
                        adminOnly: true,
                    },
                    user_management: {
                        labelKey: "userManagement",
                        routePath: "/admin/console/user-management",
                        icon: Users,
                        privileges: [],
                        adminOnly: true,
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

export type RouteAccess = {
    isAdmin: boolean;
    granted: readonly string[] | null;
};

function permitted(node: RouteNode, access: RouteAccess) {
    if (node.privileges.length === 0 || access.granted === null) return true;
    return node.privileges.some((privilege) => access.granted?.includes(privilege));
}

export function isRouteVisible(node: RouteNode, access: RouteAccess): boolean {
    if (node.hidden) return false;
    if (node.adminOnly && !access.isAdmin) return false;

    const childVisible = Object.values(node.children ?? {}).some((child) => isRouteVisible(child, access));

    if (!node.routePath) return childVisible;

    return permitted(node, access) || childVisible;
}
