"use client";

import { useActionState, useState } from "react";

import { Checkbox, Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { Card, ColorSwatch, LinkButton } from "@/components/ui";
import { pixelColor, toDateTimeLocal } from "@/lib/format";
import { emptyForm, type FormAction } from "@/lib/form-state";
import type { Campaign } from "@/lib/types";

const hex = (n: number) => n.toString(16).padStart(2, "0");

export function CampaignForm({
  campaign,
  defaultExpire,
  save,
}: {
  campaign?: Campaign;
  defaultExpire: string;
  save: FormAction;
}) {
  const [state, action] = useActionState(save, emptyForm);
  const v = state.values;
  const f = state.fields ?? {};

  const initial = (key: "pixelColorR" | "pixelColorG" | "pixelColorB" | "pixelColorA", fallback: number) =>
    v?.[key] !== undefined ? Number(v[key]) : (campaign?.[key] ?? fallback);
  const [rgb, setRgb] = useState({ r: initial("pixelColorR", 0), g: initial("pixelColorG", 0), b: initial("pixelColorB", 0) });
  const [alpha, setAlpha] = useState(initial("pixelColorA", 0));

  const colorError = f.pixelColorR ?? f.pixelColorG ?? f.pixelColorB ?? f.pixelColorA;
  const color = { pixelColorR: rgb.r, pixelColorG: rgb.g, pixelColorB: rgb.b, pixelColorA: alpha };

  return (
    <Card className="max-w-xl">
      <form action={action} className="space-y-5 p-6">
        <FormError state={state} />
        <Field label="Nome" name="name" error={f.name} hint="Único dentro do tenant.">
          <TextInput name="name" required maxLength={200} defaultValue={v?.name ?? campaign?.name} error={f.name} />
        </Field>
        <Field label="Expira em (UTC)" name="expireInUtc" error={f.expireInUtc} hint="Depois desta data o pixel deixa de contar visualizações.">
          <TextInput
            name="expireInUtc"
            type="datetime-local"
            required
            defaultValue={v?.expireInUtc ?? (campaign ? toDateTimeLocal(campaign.expireInUtc) : defaultExpire)}
            error={f.expireInUtc}
          />
        </Field>

        <fieldset>
          <legend className="block text-sm font-medium text-zinc-800">Cor do pixel</legend>
          <p className="mt-0.5 text-xs text-zinc-500">
            O PNG de 1×1 devolvido nos e-mails. Transparente (opacidade 0) é o mais discreto.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-4">
            <ColorSwatch color={pixelColor(color)} size="h-10 w-10" />
            <label className="flex items-center gap-2 text-sm text-zinc-700">
              Cor
              <input
                type="color"
                value={`#${hex(rgb.r)}${hex(rgb.g)}${hex(rgb.b)}`}
                onChange={(e) => {
                  const n = parseInt(e.target.value.slice(1), 16);
                  setRgb({ r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 });
                }}
                className="h-9 w-12 cursor-pointer rounded border border-zinc-300 bg-white p-0.5"
              />
            </label>
            <label className="flex flex-1 items-center gap-2 text-sm text-zinc-700">
              Opacidade
              <input
                type="range"
                min={0}
                max={1}
                step={0.01}
                value={alpha}
                onChange={(e) => setAlpha(Number(e.target.value))}
                className="min-w-24 flex-1 accent-indigo-600"
              />
              <span className="w-10 text-right font-mono text-xs tabular-nums">{Math.round(alpha * 100)}%</span>
            </label>
          </div>
          <p className="mt-2 font-mono text-xs text-zinc-500">
            rgba({rgb.r}, {rgb.g}, {rgb.b}, {alpha})
          </p>
          {colorError && <p className="mt-1.5 text-sm text-red-600">{colorError}</p>}
          <input type="hidden" name="pixelColorR" value={rgb.r} />
          <input type="hidden" name="pixelColorG" value={rgb.g} />
          <input type="hidden" name="pixelColorB" value={rgb.b} />
          <input type="hidden" name="pixelColorA" value={alpha} />
        </fieldset>

        <Checkbox
          name="active"
          label="Ativa"
          description="Campanhas inativas respondem 404 no pixel e não contam visualizações."
          defaultChecked={v ? v.active === "on" : (campaign?.active ?? true)}
        />
        <div className="flex gap-2 pt-2">
          <SubmitButton>{campaign ? "Salvar" : "Criar campanha"}</SubmitButton>
          <LinkButton href={campaign ? `/campaigns/${campaign.id}` : "/campaigns"} variant="secondary">
            Cancelar
          </LinkButton>
        </div>
      </form>
    </Card>
  );
}
