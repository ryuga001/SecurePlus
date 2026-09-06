"use client";

import PageTabs from "@/components/dashboard/tabs/page-tabs";

interface ConsolepageProps {
  heading: string;
  subheading?: string;
  actions?: React.ReactNode;
  data: React.ReactNode;
  footer?: React.ReactNode;
}

const Consolepage = ({ heading, subheading, actions, data, footer }: ConsolepageProps) => {
  return (
    <section className="flex flex-col gap-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">{heading}</h1>
          {subheading ? <p className="mt-1 text-sm text-muted-foreground">{subheading}</p> : null}
        </div>
        {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
      </header>

      <PageTabs />

      <main>{data}</main>

      {footer ? <footer>{footer}</footer> : null}
    </section>
  );
};

export default Consolepage;
