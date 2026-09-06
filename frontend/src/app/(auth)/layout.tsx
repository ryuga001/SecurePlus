import Image from "next/image";
import Link from "next/link";
import { Activity, LockKeyhole, ShieldCheck } from "lucide-react";

const highlights = [
  {
    icon: ShieldCheck,
    title: "Continuous assurance",
    description: "Posture, policy and access checks across every connected system.",
  },
  {
    icon: Activity,
    title: "Live threat signal",
    description: "Incidents, alerts and audit trails the moment they happen.",
  },
  {
    icon: LockKeyhole,
    title: "Secure by default",
    description: "Encrypted records, scoped roles, verified sign-in every session.",
  },
];

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid h-svh overflow-hidden lg:grid-cols-[1.05fr_1fr]">
      {/* Brand / illustration panel */}
      <div className="relative hidden overflow-hidden bg-primary lg:flex lg:flex-col">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,oklch(0.72_0.16_250/0.65),transparent_55%),radial-gradient(circle_at_bottom_left,oklch(0.35_0.16_268/0.85),transparent_60%)]" />
        <div
          aria-hidden
          className="absolute inset-0 opacity-[0.14] [background-image:linear-gradient(to_right,white_1px,transparent_1px),linear-gradient(to_bottom,white_1px,transparent_1px)] [background-size:56px_56px]"
        />

        <div className="relative flex min-h-0 flex-1 flex-col gap-6 p-8 xl:gap-8 xl:p-12">
          <Link href="/" className="flex w-fit shrink-0 items-center gap-3">
            <span className="flex size-10 items-center justify-center bg-white/15 ring-1 ring-white/25 backdrop-blur">
              <ShieldCheck className="size-5.5 text-white" />
            </span>
            <span className="text-xl font-semibold tracking-tight text-white">SecurePlus</span>
          </Link>

          <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-6">
            <div className="relative min-h-0 w-full max-w-xl flex-1 bg-white/10 p-2 ring-1 ring-white/20 shadow-2xl shadow-blue-950/40 backdrop-blur">
              <Image
                src="/auth-illustration.jpeg"
                alt="Illustration of a security dashboard with access records and protection controls"
                fill
                sizes="(min-width: 1024px) 45vw, 0px"
                priority
                className="object-contain p-2"
              />
            </div>

            <div className="max-w-xl shrink-0 space-y-2 text-center">
              <h2 className="text-2xl font-semibold tracking-tight text-white xl:text-3xl">
                Security, under control
              </h2>
              <p className="text-sm leading-relaxed text-blue-50/80 xl:text-base">
                Monitor access, investigate incidents and keep compliance evidence audit-ready —
                from one secure console.
              </p>
            </div>
          </div>

          <ul className="grid shrink-0 gap-3 sm:grid-cols-3">
            {highlights.map(({ icon: Icon, title, description }) => (
              <li key={title} className="bg-white/10 p-3.5 ring-1 ring-white/15 backdrop-blur">
                <Icon className="size-5 text-white" />
                <p className="mt-2.5 text-sm font-medium text-white">{title}</p>
                <p className="mt-1 text-xs leading-relaxed text-blue-50/70">{description}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Form panel — scrolls inside itself, never the page */}
      <div className="relative flex h-svh flex-col overflow-y-auto bg-background">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-x-0 top-0 h-64 bg-gradient-to-b from-primary/8 to-transparent"
        />

        <div className="relative flex min-h-full flex-1 items-center justify-center px-5 py-8 sm:px-8">
          <div className="w-full max-w-md">
            <Link href="/" className="mb-8 flex w-fit items-center gap-3 lg:hidden">
              <span className="flex size-10 items-center justify-center bg-primary">
                <ShieldCheck className="size-5.5 text-primary-foreground" />
              </span>
              <span className="text-xl font-semibold tracking-tight">SecurePlus</span>
            </Link>

            {children}

            <p className="mt-8 text-center text-xs text-muted-foreground">
              Protected by verified email sign-in · Encrypted end to end
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
