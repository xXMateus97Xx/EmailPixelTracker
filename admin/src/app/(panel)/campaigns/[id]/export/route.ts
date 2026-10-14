import { type NextRequest, NextResponse } from "next/server";

import { ApiError, download } from "@/lib/api";
import { reportPeriod } from "../period";

/**
 * Downloads every report of the campaign as an Excel workbook. The API builds
 * the file; it is streamed through, so the browser never needs the API token.
 */
export async function GET(request: NextRequest, { params }: RouteContext<"/campaigns/[id]/export">) {
  const { id } = await params;
  const { from, to, query } = reportPeriod(Object.fromEntries(request.nextUrl.searchParams));
  const back = new URL(`/campaigns/${encodeURIComponent(id)}`, request.url);
  back.searchParams.set("from", from);
  back.searchParams.set("to", to);

  let res: Response;
  try {
    res = await download(`/api/campaigns/${encodeURIComponent(id)}/reports/export`, query);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      return NextResponse.redirect(new URL("/login?reason=expired", request.url), 303);
    }
    // Back to the report, which explains the failure.
    console.error("export failed", err);
    back.searchParams.set("exportError", err instanceof ApiError && err.requestId ? err.requestId : "1");
    return NextResponse.redirect(back, 303);
  }

  const headers = new Headers({ "Cache-Control": "no-store" });
  for (const name of ["Content-Type", "Content-Disposition", "Content-Length"]) {
    const value = res.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }
  return new Response(res.body, { headers });
}
