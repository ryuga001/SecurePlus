import {
    FileText,
    LayoutDashboard,
    Mail,
    Regex,
    ScrollText,
    Settings,
    ShieldCheck,
    SlidersHorizontal,
    Users,
    UsersRound,
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
                routePath: "/admin/email/configurations",
                icon: SlidersHorizontal,
                privileges: [],
                heading: "Email configurations",
                subheading: "Connected mailboxes, gateways and delivery settings.",
            },
            policies: {
                label: "Policies",
                routePath: "/admin/policies",
                icon: ShieldCheck,
                privileges: [],
                heading: "Email policies",
                subheading: "Which rules apply to which groups of users.",
            },
            rules: {
                label: "Rules",
                routePath: "/admin/rules",
                icon: Regex,
                privileges: [],
                heading: "Rules",
                subheading: "Keyword and regular-expression matchers that policies apply to mail.",
            },
            users: {
                label: "Users",
                routePath: "/admin/email/users",
                icon: Users,
                privileges: [],
                heading: "Email users",
                subheading: "The mailboxes this workspace protects.",
            },
            groups: {
                label: "Groups",
                routePath: "/admin/email/groups",
                icon: UsersRound,
                privileges: [],
                heading: "Email groups",
                subheading: "Collections of users that policies target together.",
            },
            reports: {
                label: "Reports",
                routePath: "/email/reports",
                icon: FileText,
                privileges: [],
                heading: "Email reports",
                subheading: "Exportable evidence for audits and reviews.",
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