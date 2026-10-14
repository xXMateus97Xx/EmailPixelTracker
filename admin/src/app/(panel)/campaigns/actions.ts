"use server";

import { redirect } from "next/navigation";

import { api } from "@/lib/api";
import type { FormState } from "@/lib/form-state";
import { errorState, formValues } from "@/lib/forms";
import type { Campaign } from "@/lib/types";

/** Creates (id undefined) or updates a campaign of the current tenant. */
export async function saveCampaign(id: string | undefined, _: FormState, formData: FormData): Promise<FormState> {
  const values = formValues(formData);
  const number = (key: string) => Number(values[key] ?? "0");
  const body = {
    name: values.name ?? "",
    active: formData.get("active") === "on",
    // The form edits the date in UTC (datetime-local has no time zone);
    // an empty value is omitted so the API reports it as required.
    expireInUtc: values.expireInUtc ? `${values.expireInUtc}:00Z` : undefined,
    pixelColorR: number("pixelColorR"),
    pixelColorG: number("pixelColorG"),
    pixelColorB: number("pixelColorB"),
    pixelColorA: number("pixelColorA"),
  };

  let saved: Campaign;
  try {
    saved = id
      ? await api<Campaign>("PUT", `/api/campaigns/${encodeURIComponent(id)}`, { body })
      : await api<Campaign>("POST", "/api/campaigns", { body });
  } catch (err) {
    return errorState(err, { ...values, active: body.active ? "on" : "" });
  }
  redirect(`/campaigns/${saved.id}`);
}
