"use server";

import { redirect } from "next/navigation";

import { api } from "@/lib/api";
import type { FormState } from "@/lib/form-state";
import { errorState, formValues } from "@/lib/forms";

/** Creates (id undefined) or updates a tenant. */
export async function saveTenant(id: string | undefined, _: FormState, formData: FormData): Promise<FormState> {
  const values = formValues(formData);
  const body = { name: values.name ?? "", active: formData.get("active") === "on" };
  try {
    if (id) {
      await api("PUT", `/api/tenants/${encodeURIComponent(id)}`, { body });
    } else {
      await api("POST", "/api/tenants", { body });
    }
  } catch (err) {
    return errorState(err, { ...values, active: body.active ? "on" : "" });
  }
  redirect("/tenants");
}
