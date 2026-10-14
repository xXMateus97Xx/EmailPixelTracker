"use server";

import { redirect } from "next/navigation";

import { api } from "@/lib/api";
import type { FormState } from "@/lib/form-state";
import { errorState, formValues } from "@/lib/forms";

/** Creates (id undefined) or updates a user of the current scope. */
export async function saveUser(id: string | undefined, _: FormState, formData: FormData): Promise<FormState> {
  const values = formValues(formData);
  const body = {
    name: values.name ?? "",
    username: values.username ?? "",
    // On update an empty password keeps the current one.
    password: String(formData.get("password") ?? ""),
    active: formData.get("active") === "on",
    role: values.role ?? "",
  };
  try {
    if (id) {
      await api("PUT", `/api/users/${encodeURIComponent(id)}`, { body });
    } else {
      await api("POST", "/api/users", { body });
    }
  } catch (err) {
    return errorState(err, { ...values, active: body.active ? "on" : "" });
  }
  redirect("/users");
}
