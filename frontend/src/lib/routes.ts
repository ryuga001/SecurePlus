import {
    FileText,
    LayoutDashboard,
    Mail,
    ScrollText,
    Settings,
    ShieldCheck,
    SlidersHorizontal,
    Users,
    type LucideIcon,
} from "lucide-react";

export type RouteNode = {
    label: string;
    routePath?: string;
    icon?: LucideIcon;
    privileges: readonly string[];
    heading?: string;
    subheading?: string;
    hidden?: boolean;
    children?: Record<string, RouteNode>;
};

export type RouteMap = Record<string, RouteNode>;

export const routes = {
    dashboard: {
        label: "Dashboard",
        routePath: "/dashboard",
        icon: LayoutDashboard,
        privileges: [],
        heading: "Security overview",
        subheading: "Live posture, recent incidents and outstanding actions.",
    },

    emailProtection: {
        label: "Email Protection",
        icon: Mail,
        privileges: [],
        children: {
            configurations: {
                label: "Configurations",
                routePath: "/email/configurations",
                icon: SlidersHorizontal,
                privileges: [],
                heading: "Email configurations",
                subheading: "Connected mailboxes, gateways and delivery settings.",
            },
            policy: {
                label: "Policy",
                routePath: "/email/policy",
                icon: ShieldCheck,
                privileges: [],
                heading: "Email policy",
                subheading: "Rules that decide how inbound and outbound mail is handled.",
            },
            reports: {
                label: "Reports",
                routePath: "/email/reports",
                icon: FileText,
                privileges: [],
                heading: "Email reports",
                subheading: "Exportable evidence for audits and reviews.",
            },
            users: {
                label: "Users",
                routePath: "/email/users-console",
                icon: Users,
                privileges: [],
                heading: "Users console",
                subheading: "Manage and configure user access and permissions.",
            }
        },
    },
    admin_console: {
        label: "Admin Console",
        routePath: "/admin/console",
        icon: ScrollText,
        privileges: [],
        heading: "Admin console",
        subheading: "Manage and configure your organisation.",
        children: {
            user_roles: {
                label: "User Roles",
                routePath: "/admin/console/user-roles",
                icon: Users,
                privileges: [],
            },
            user_management: {
                label: "User Management",
                routePath: "/admin/console/user-management",
                icon: Users,
                privileges: [],
            },
        }
    },
    settings: {
        label: "Settings",
        routePath: "/admin/settings",
        icon: Settings,
        privileges: [],
        heading: "Organisation settings",
        subheading: "Profile, security defaults and retention windows.",
    },
} as const satisfies RouteMap;