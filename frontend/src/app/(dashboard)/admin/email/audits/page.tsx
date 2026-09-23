import { redirect } from "next/navigation";

export default function EmailAuditsPage() {
  redirect("/admin/email/audits/incidents");
}
