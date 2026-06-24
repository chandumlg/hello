# Daily Markets Catalyst & News Brief
**Run timestamp:** 2026-06-24T0012Z (UTC) · Horizon: 1 day – 2 weeks
*News-flow characterization only. No buy/sell calls, no price predictions. You make the decisions.*

---

## CHANGES SINCE LAST RUN

**First run — no prior state.** The `state/` directory was empty, so there is no prior snapshot to diff against. Everything below is treated as newly seen; tag transitions, resolved-event tracking, and staleness aging begin from this run. Elapsed time since prior run: N/A.

A few items that the *next* run will likely flag as resolved/transitioned:
- **MU (Micron)** reports fiscal Q3 2026 **today, June 24, after the close** (~00:30Z June 25). Tagged EARNINGS IMMINENT; next run should resolve it with the actual print and reaction.
- **TSLA** Q2 production/delivery report due **~July 1–3** — a hard catalyst inside the 14-day window (not an earnings print, so no EARNINGS IMMINENT tag, but high-attention).
- Macro: **PCE (~Jun 25)** and **June jobs report (~Jul 2)** land inside the window; next **FOMC decision is Jul 29** (just outside).

---

## PER-TICKER BRIEF

See the JSON array below for the structured per-ticker breakdown (scan_tag, new_today, upcoming_events, bull/bear factors, read_confidence).

```json
[
  {
    "ticker": "GOOGL",
    "name": "Alphabet/Google",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — Worst day in >1yr (~-5%) on AI talent exits: Gemini co-lead Noam Shazeer to OpenAI; DeepMind's John Jumper (Nobel laureate) to Anthropic. https://www.cnbc.com/2026/06/22/alphabet-goog-stock-ai-departures.html",
      "2026-06-22 — CA judge denied Google/YouTube (and Meta) a new trial after youth-addiction jury verdict; UK pushing under-16 social restrictions hitting YouTube. https://www.timothysykes.com/news/alphabet-inc-googl-news-2026_06_22/",
      "~2026-06-15 — $1.5B Alabama data-center expansion (2026–2027). https://www.insidermonkey.com/blog/alphabet-googl-announces-1-5-billion-investment-across-2026-and-2027-to-expand-its-alabama-data-center-campus-1784545/"
    ],
    "still_relevant": [
      "2026-06-01 — $80B equity raise to fund AI buildout, incl. confirmed $10B Berkshire private placement; 2026 capex guided $180–190B. first_seen 2026-06-24T0012Z. https://www.cnbc.com/2026/06/01/alphabet-to-raise-80-billion-from-stock-sales-to-fund-ai-buildout.html",
      "Ongoing — DOJ + states appealed the search-remedy ruling (no Chrome/Android divestiture; ends exclusive defaults). first_seen 2026-06-24T0012Z. https://finance.yahoo.com/news/us-government-appeals-remedy-google-120633436.html"
    ],
    "upcoming_events": "None confirmed in window. Q2 2026 earnings expected ~Jul 22–28 (outside window, date unconfirmed).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Scale of AI capex ($180–190B) + Berkshire $10B endorsement; antitrust outcome avoided structural breakup; broad Buy consensus with some PT raises.",
    "factors_that_could_push_it_down": "High-profile AI talent defections (retention/competitive worry); regulatory/legal stack (DOJ appeal, youth-addiction ruling, UK under-16 rules threatening ad/engagement); capex + dilution from the raise.",
    "read_confidence": "high"
  },
  {
    "ticker": "META",
    "name": "Meta Platforms",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-23 — Meta Glasses launch (26 styles); Threads hits 500M MAU + new Communities features. https://about.fb.com/news/",
      "2026-06-16 — New AI-powered Facebook features; expansion of Instagram Live Video Ads / live-commerce partners. https://about.fb.com/news/",
      "2026-06-22 — CA judge denied Meta (with Google/YouTube) a new trial after youth-addiction jury verdict. https://www.timothysykes.com/news/alphabet-inc-googl-news-2026_06_22/"
    ],
    "still_relevant": [
      "Ongoing — EU DSA child-protection preliminary findings (potential fine up to ~6% of global turnover) + separate EU antitrust probe into WhatsApp AI integration (up to ~10% of revenue). Amounts UNCONFIRMED. first_seen 2026-06-24T0012Z. https://www.tikr.com/blog/meta-faces-a-multibillion-dollar-eu-fine-risk-heres-what-it-means-for-the-stock-at-17x-earnings",
      "Ongoing — Aggressive Superintelligence Labs hiring (50+ researchers poached; reports of a pause). first_seen 2026-06-24T0012Z. https://stocktwits.com/news-articles/markets/equity/that-s-it-for-now-meta-halts-ai-hiring-after-poaching-over-50-with-multi-million-offers-report/chsS0Z1RdLe"
    ],
    "upcoming_events": "None confirmed in window. Q2 2026 earnings expected ~Jul 29 (outside window, date unconfirmed).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Strong product cadence (Glasses, Threads 500M MAU, AI features, live-commerce ad formats); Q1 EPS beat ($7.31 vs $6.67); broad Buy consensus.",
    "factors_that_could_push_it_down": "Material EU regulatory tail risk (DSA + WhatsApp AI probes, multi-billion potential); US youth-addiction litigation; heavy AI capex + costly talent war on the spend narrative.",
    "read_confidence": "medium-high"
  },
  {
    "ticker": "TSLA",
    "name": "Tesla",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — NHTSA opened special crash investigation into fatal Katy, TX Model 3 crash; stock fell ~6%. https://www.cnbc.com/2026/06/22/tesla-nhtsa-model-3-crash-autopilot-katy-texas.html",
      "2026-06-23 — Tesla admitted FSD was engaged in the fatal crash; sits atop a 3.2M-vehicle FSD Engineering Analysis (last step before possible recall) + separate 2.88M-vehicle probe. https://electrek.co/2026/06/23/tesla-fsd-katy-crash-driver-pedal/",
      "2026-06-22/23 — Jefferies cut PT to $375, floated 'SpaceX proxy'/merger speculation (merger talk UNCONFIRMED). https://stocktwits.com/news-articles/markets/equity/tesla-red-june-jefferies-spacex-proxy-merger-talk/cZKvq2WR79j",
      "2026-06-16/17 — Musk Form 4: exercised ~304M options from 2018 package (strike $23.34; ~$116B paper gain). https://www.satonic-autoparts.com/blogs/news/tesla-news-update-june-2026",
      "~2026-06-08+ — Robotaxi: Texas commercial driverless authorization, Level-4 self-cert, Austin metro-wide geofence; Bloomberg notes fleet only ~59 vehicles. https://www.bloomberg.com/news/features/2026-06-10/tesla-robotaxi-fleet-totals-just-59-vehicles-despite-musk-promises",
      "Mid-June — Goldman raised Q2 delivery forecast to 420K (from 405K); RBC/UBS/Goldman see >400K. https://www.basenor.com/blogs/news/goldman-sachs-raises-tesla-q2-2026-delivery-forecast-to-420k"
    ],
    "still_relevant": "none",
    "upcoming_events": "~Jul 1–3 — Q2 2026 production & delivery report (within window; high-attention). Q2 earnings ~Jul 22 (outside window, unconfirmed). https://ir.tesla.com/press",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Rising Q2 delivery estimates (Goldman 420K) on strong Europe/China; robotaxi regulatory progress; NatPower battery-storage partnership (>$15B projected).",
    "factors_that_could_push_it_down": "New NHTSA fatal-crash probe + FSD-engaged admission → recall risk on 3.2M vehicles; Jefferies PT cut; ~7% June decline; tiny actual robotaxi fleet (~59) vs promises; Optimus behind schedule.",
    "read_confidence": "high"
  },
  {
    "ticker": "AVGO",
    "name": "Broadcom",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — Ex-dividend ($0.65/sh quarterly; pay Jun 30); Strong-Buy consensus, avg PT ~$522–525. https://stockanalysis.com/stocks/avgo/dividend/"
    ],
    "still_relevant": [
      "2026-06-03 — Q2 FY26: EPS $2.44 vs $2.32; rev $22.2B (+48% YoY); AI semi +143% to $10.8B — BUT CEO did not raise >$100B FY26 AI target; Q3 AI guide ~$16B missed ~$17.2B; stock fell ~15–22% on the week, triggering a sector sell-off. first_seen 2026-06-24T0012Z. https://www.cnbc.com/2026/06/03/broadcom-avgo-earnings-report-q2-2026.html",
      "Ongoing — OpenAI/Broadcom 10GW custom-accelerator collaboration (deploy H2 2026→2029). first_seen 2026-06-24T0012Z. https://investors.broadcom.com/news-releases/news-release-details/openai-and-broadcom-announce-strategic-collaboration-deploy-10"
    ],
    "upcoming_events": "Dividend pay date Jun 30. No earnings in window (Q3 FY26 typically early September).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Record Q2 rev/FCF; AI semi +143% YoY; six custom-silicon customers (incl. Google, Meta, OpenAI, Anthropic); Strong-Buy consensus, PTs to $650.",
    "factors_that_could_push_it_down": "No-raise on AI target disappointed; Q3 AI guide below whisper; sharp post-earnings drawdown; named as the catalyst for the broad chip sell-off; rate/AI-sentiment sensitivity.",
    "read_confidence": "high"
  },
  {
    "ticker": "NVDA",
    "name": "Nvidia",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — ISC (Hamburg) cluster: Vera Rubin for scientific supercomputing; 35 new NVIDIA AI supercomputers across Europe; 'Halos for Robotics' safety system. https://www.globenewswire.com/news-release/2026/06/22/3315318/0/en/NVIDIA-Vera-Rubin-Delivers-World-Class-Supercomputers-for-Science.html",
      "2026-06-22 — Stock -0.97% to ~$208.65 amid an AI-stock sell-off hitting NVDA/AMD/MU. https://www.cnbc.com/quotes/NVDA"
    ],
    "still_relevant": [
      "Ongoing — China export-control overhang: H20 'green-zone' chips can flow; H200 China sales stalled, NO China revenue yet, import approval uncertain. Treat 'shipments resumed' as UNCONFIRMED. first_seen 2026-06-24T0012Z. https://finance.yahoo.com/news/nvidia-still-hasnt-finalized-deal-to-kick-15-of-h20-china-chip-sales-back-to-the-us-government-230229161.html"
    ],
    "upcoming_events": "None major in window. Next earnings ~Aug 26 (outside window).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Deployment pipeline (OpenAI 10GW LOI, Meta multi-year, IREN up to 5GW); Vera Rubin ramp; CPU TAM expansion; Strong-Buy consensus.",
    "factors_that_could_push_it_down": "China H200 revenue still zero / approval uncertain; trading well below 52-wk high; caught in AI sell-off; 'stretched valuation' framing; hyperscaler capex-digestion risk.",
    "read_confidence": "medium-high"
  },
  {
    "ticker": "INTC",
    "name": "Intel",
    "scan_tag": "POSITIVE FLOW",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-18 — Shares +9–10.5% to a record close (~$134) after Trump (Truth Social) said Apple will work with Intel to design/make chips in the US. Deal terms NOT fully company-confirmed — scope partially UNCONFIRMED. https://www.cnbc.com/2026/06/18/trump-intel-apple-chip-design-deal.html",
      "2026-06-16 — Intel 18A-P entered risk production; CEO cited ~7% monthly yield improvement on 18A, ahead of targets. https://newsroom.intel.com/intel-foundry/intel-foundry-details-process-milestones-future-innovation-at-vlsi-symposium",
      "~2026-06-22/23 — Mizuho raised PT to $135 (Neutral); consensus still 'Hold', avg PT ~$79 (below market). https://www.marketbeat.com/stocks/NASDAQ/INTC/forecast/"
    ],
    "still_relevant": [
      "Ongoing — US government 10% stake (~433M sh @ $20.47, ~$8.9B) now worth $40B+ on the rally. first_seen 2026-06-24T0012Z. https://finance.yahoo.com/markets/stocks/articles/intel-stock-190-2026-trump-131501491.html"
    ],
    "upcoming_events": "None in window. Q2 2026 earnings Jul 23 (just outside window).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Apple design/manufacturing partnership (TSMC diversification); record share price; 18A/Panther Lake traction + Kontron edge-AI win; US government backing; Mizuho PT raise.",
    "factors_that_could_push_it_down": "Apple deal largely Trump-announced, not fully company-confirmed; consensus still 'Hold' with avg PT (~$79) well below market (valuation may price near-perfect execution); restructuring/operating-loss/competition risk.",
    "read_confidence": "medium-high"
  },
  {
    "ticker": "MU",
    "name": "Micron",
    "scan_tag": "EARNINGS IMMINENT",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-24 (TODAY, after close) — Fiscal Q3 2026 earnings. Guidance: record rev ~$33.5B ±$0.75B, GM ~81%, non-GAAP EPS $19.15 ±$0.40; consensus ~$34.5B / ~$19.72. https://www.stocktitan.net/news/MU/micron-technology-to-report-fiscal-third-quarter-results-on-june-24-22gcrbths4gp.html",
      "2026-06-22 — Wave of large PT increases into the print (TD Cowen→$1,500, UBS→$1,625, Needham $1,550, Stifel $1,500, others $1,200–1,600), thesis: HBM 'structurally durable' / 2026 HBM sold out. https://www.fool.com/investing/2026/06/22/prediction-micron-stock-will-skyrocket-after-june/"
    ],
    "still_relevant": "none",
    "upcoming_events": "Fiscal Q3 2026 earnings + call: Jun 24 (today, after close).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "HBM capacity sold out for 2026; data-center memory framed as structural; record guidance (~81% GM); sharply raised Street PTs.",
    "factors_that_could_push_it_down": "Very high expectations into the print = setup risk; memory output redirected to data center straining smartphone/PC supply (mix/cyclicality worry); caught in broad AI-chip sell-off; elevated valuation.",
    "read_confidence": "high"
  },
  {
    "ticker": "VRT",
    "name": "Vertiv",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-23 — Stock fell ~8% intraday in an AI/semi sector sell-off (Korea chip names; regulator remarks). Macro-driven. https://www.aol.com/articles/heres-why-shares-vertiv-crashed-180615046.html",
      "2026-06-18 — GLJ Research upgraded VRT to Hold from Sell (CoolIt 15kW cold-plate extends single-phase liquid-cooling runway); Bernstein reiterated Buy in June. https://finance.yahoo.com/markets/stocks/articles/coolit-innovation-supports-vertiv-vrt-024141257.html",
      "2026-06-12 — Closed ThermoKey S.p.A. acquisition (thermal/heat-rejection, EMEA capacity). https://www.insidermonkey.com/blog/why-vertiv-vrt-is-moving-deeper-across-the-full-ai-data-center-thermal-chain-1783751/"
    ],
    "still_relevant": "none",
    "upcoming_events": "Jun 25 — dividend payment (declared). No earnings in window; Q2 2026 ~late Jul/early Aug (date unconfirmed).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Bullish analyst consensus (mean PT ~$377, high ~$500); prior bear turning less negative; ~$15B order backlog tied to AI buildout; end-to-end power+cooling positioning via ThermoKey + partnerships.",
    "factors_that_could_push_it_down": "Premium valuation (P/E cited ~82x vs ~40x industry) → vulnerable to multiple compression/profit-taking; high beta to AI-trade sentiment. UNVERIFIED single-source bear note of a soft Q2 (couldn't confirm vs filing).",
    "read_confidence": "medium"
  },
  {
    "ticker": "CEG",
    "name": "Constellation Energy",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-23 — Walmart long-term nuclear PPA: ~176 MW from Dresden (incl. ~30 MW expansion), two 15-yr terms beginning 2029/2030; Walmart's first nuclear PPA. https://corporate.walmart.com/news/2026/06/23/constellation-and-walmart-announce-longterm-agreement-to-support-reliable-emissionsfree-nuclear-energy-in-illinois",
      "2026-06-17 — Bernstein initiated Outperform, $296 PT. https://www.streetinsider.com/Analyst+Comments/Bernstein+SocGen+Group+Starts+Constellation+Energy+(CEG)+at+Outperform/26654418.html",
      "2026-06-16 — Goldman initiated Neutral, $305 PT (valuation; prefers Talen/Vistra/NRG). https://www.investing.com/news/analyst-ratings/goldman-sachs-initiates-constellation-energy-stock-at-neutral-on-valuation-93CH-4749288"
    ],
    "still_relevant": [
      "2026-06-02 — Secondary offering closed: 11.0M sh @ $281 (~$3.1B) sold by Calpine-deal holders (no new CEG shares); CEG repurchased 2.0M sh ($558M); underwriter 30-day option (up to 1.35M sh) = near-term supply overhang. first_seen 2026-06-24T0012Z. https://www.tipranks.com/news/company-announcements/constellation-energy-completes-secondary-offering-and-share-repurchase",
      "Ongoing — Meta 20-yr ~1.1GW Clinton nuclear PPA (offtake from Jun 2027); 5,650+ MW of long-term clean-energy deals. first_seen 2026-06-24T0012Z. https://finance.yahoo.com/sectors/energy/articles/ai-data-center-power-deals-181559704.html"
    ],
    "upcoming_events": "No company event in window. Q2 2026 earnings ~Jul 30 (outside window, unconfirmed). Underwriter 30-day secondary option could be exercised into early July (mechanical).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "New Walmart 15-yr nuclear PPA adds to premium long-dated PPA pattern (Meta/Microsoft-TMI/CyrusOne); structural AI/data-center power demand (PJM capacity prices surged); Calpine scale; ~$3.5B buyback firepower.",
    "factors_that_could_push_it_down": "Stock down ~26% YTD; targets trimmed; Goldman Neutral and new PTs only modestly above price; large-holder secondary overhang + underwriter option; earlier heavy insider selling; data-center deal competition (Talen–Amazon).",
    "read_confidence": "medium-high"
  },
  {
    "ticker": "AMZN",
    "name": "Amazon",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-23 to 06-26 — Prime Day 2026 LIVE (expanded 4-day, moved to June). https://www.aboutamazon.com/news/retail/amazon-prime-day-2026-date",
      "2026-06-18 — Preliminary talks to sell Trainium AI chips externally (no customers named — UNCONFIRMED as a transaction); custom silicon ~$20B run-rate. https://www.bloomberg.com/news/articles/2026-06-18/amazon-is-in-talks-to-sell-nvidia-rival-chips-to-other-companies",
      "2026-06-16 — Reported potential FTC advertising lawsuit (possible billions in penalties); suit NOT yet filed — UNCONFIRMED. https://news.bloomberglaw.com/antitrust/amazon-faces-billions-in-penalties-from-potential-ftc-ad-suit"
    ],
    "still_relevant": [
      "Early–mid June — Record debt financing for ~$200B 2026 capex (C$14B Canadian bond, €14.5B euro deal, $17.5B credit line). first_seen 2026-06-24T0012Z. https://www.briefs.co/news/amazon-record-c14-billion-canadian-bond-sale-ai/"
    ],
    "upcoming_events": "Jun 23–26 — Prime Day (in progress). Q2 2026 earnings Jul 30 AMC (outside window). FTC ad-suit resolution 'as soon as this summer' — no firm date.",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "AWS momentum (Q1 +28% YoY to $37.6B); expanded OpenAI (+$100B/8yr, ~2GW Trainium) & Anthropic (up to 5GW) deals; custom-silicon optionality (~$20B run-rate); $50B AWS federal AI build-out; Prime Day live.",
    "factors_that_could_push_it_down": "Regulatory overhang (potential FTC ad suit + 2027 marketplace antitrust trial); ~$200B 2026 capex (highest in Mag 7) partly debt-funded; workforce/headline risk (30,000+ cuts, data-center pushback); Trainium external sales unproven vs Nvidia.",
    "read_confidence": "medium"
  },
  {
    "ticker": "PLTR",
    "name": "Palantir",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — Secured foundational role in US Army NGC2 (Foundry as core cloud data layer), alongside Anduril; Army established NGC2 baseline. https://finance.yahoo.com/technology/ai/articles/palantir-secures-foundational-role-ngc2-200500839.html",
      "2026-06-22 — Stock hit a new 52-week low (~-7% intraday) on valuation scrutiny + European headwinds. https://www.tradingkey.com/news/market-movers/261982215-market-movers-pltr-20260622",
      "2026-06-16 — France's DGSI to replace Palantir with ChapsVision (digital-sovereignty); UK MPs pressing NHS FDP contract (Feb 2027 break clause). https://www.bloomberg.com/news/articles/2026-06-16/french-security-service-to-replace-palantir-with-local-software"
    ],
    "still_relevant": [
      "2026-05-04 — Q1 2026 beat (EPS $0.33 vs $0.28; rev $1.63B; ~85% YoY) and raised FY26 guide to $7.65–7.66B. first_seen 2026-06-24T0012Z. https://www.cnbc.com/2026/05/04/palantir-pltr-q1-earnings-report-2026.html"
    ],
    "upcoming_events": "No confirmed company event in window (AIPCon 10 was Jun 4). Q2 2026 earnings Aug 10 AMC (outside window).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "US Army NGC2 foundational role reinforces core defense franchise (US gov rev +84% YoY in Q1); raised FY26 guide above consensus.",
    "factors_that_could_push_it_down": "European sovereignty backlash (France DGSI replacement; UK NHS contract threat); valuation scrutiny; 52-week low on Jun 22.",
    "read_confidence": "high"
  },
  {
    "ticker": "SMCI",
    "name": "Super Micro Computer",
    "scan_tag": "POSITIVE FLOW",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-22 — GF Securities upgraded SMCI to Buy ($48 PT); shares +15.7–17% (best day in >1yr), tied to NVIDIA Vera Rubin momentum. https://mlq.ai/news/super-micro-surges-17-on-nvidia-vera-rubin-partnership-and-analyst-upgrade/",
      "~2026-06-17/18 — Named global system builder / launch partner for NVIDIA Vera Rubin NVL4; unveiled next-gen liquid-cooled racks. https://finance.yahoo.com/markets/stocks/articles/supermicro-smci-joins-nvidia-vera-001347671.html"
    ],
    "still_relevant": [
      "2026-06-09 — $7.0B financing (~$1.25B common + ~$3.75B mandatory convertible preferred + up-to-$2.0B ATM from Q3); funds components for ~$39B of AI server orders. Stock fell ~19.7% intraday Jun 10 on dilution. first_seen 2026-06-24T0012Z. https://www.sec.gov/Archives/edgar/data/0001375365/000119312526269703/d45696dex991.htm",
      "2026-05-05 — Q3 FY26: rev $10.2B, NI $483M; raised Q4 guide to $11.0–12.5B, FY26 to $38.9–40.4B. first_seen 2026-06-24T0012Z. https://www.cnbc.com/2026/05/05/super-micro-smci-q3-earnings-report-2026.html"
    ],
    "upcoming_events": "No confirmed company event in window. Q4 FY26 earnings ~Aug 11 AMC (outside window, some aggregators est. Aug 4–7).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "NVIDIA Vera Rubin NVL4 launch-partner status; ~$39B AI server order backlog; first bullish rating change in months (GF Securities); raised FY26 guide.",
    "factors_that_could_push_it_down": "Recent $7.0B dilutive raise (sharp ~19.7% drop on announcement); historical accounting/governance overhang ('governance discount'); compressed margins; high beta to AI-trade sentiment.",
    "read_confidence": "high"
  },
  {
    "ticker": "SOFI",
    "name": "SoFi Technologies",
    "scan_tag": "MIXED",
    "tag_change": "unchanged",
    "new_today": [
      "2026-06-16 — CEO Anthony Noto Form 4: bought 13,888 sh (~$250,800 @ ~$18.06) — his 5th open-market buy of 2026 (~$2.25M YTD). https://www.stocktitan.net/sec-filings/SOFI/form-4-so-fi-technologies-inc-insider-trading-activity-f798db743c79.html",
      "2026-06-18 — Annual meeting: directors, pay, auditor approved. https://stockstotrade.com/news/sofi-technologies-inc-sofi-news-2026_06_22/",
      "~2026-06-17 — Named in SpaceX IPO prospectus as a potential retail broker (prospectus mention, not a formal partnership). https://www.timothysykes.com/news/sofi-technologies-inc-sofi-news-2026_06_17/"
    ],
    "still_relevant": [
      "2026-05-27 — SoFiUSD launched (first stablecoin issued by a US national bank on a banking app; ~15M members); AI tools Composer & SoFi Coach. first_seen 2026-06-24T0012Z. https://www.businesswire.com/news/home/20260527091798/en/SoFiUSD-Becomes-the-First-Stablecoin-Issued-by-a-US-National-Bank-to-Launch-on-a-Banking-Platform",
      "2026-05-12 — Truist trimmed PT to $17 from $20 (Hold) on lower Q2 revenue expectations. first_seen 2026-06-24T0012Z. https://www.tipranks.com/news/sofi-stock-gets-a-rating-downgrade-from-a-top-ranked-analyst-today"
    ],
    "upcoming_events": "No confirmed company event in window. Q2 2026 earnings ~Jul 28–29 BMO (unconfirmed; near/just past window edge).",
    "resolved_events": "none",
    "factors_that_could_push_it_up": "Persistent CEO insider buying (5 purchases in 2026, ~$2.25M YTD); product momentum (SoFiUSD to ~15M members, AI tools); SpaceX IPO retail-broker mention.",
    "factors_that_could_push_it_down": "Stock down ~40% YTD; consensus 'Hold'; Truist PT trim to $17 on softer lending/tech-platform revenue; rate-sensitivity (hawkish Fed dot plot).",
    "read_confidence": "medium-high"
  }
]
```

---

## MACRO & SECTOR CONTEXT

**Rates / Fed (hits everything; most acute for high-multiple PLTR, TSLA, SOFI, SMCI):** The **Jun 17 FOMC held at 3.50–3.75%** — Kevin Warsh's first meeting as Chair — but the **dot plot turned hawkish**, with the median 2026 year-end projection moving up to ~3.75–4.00% (implying at least one *hike* this year). Nine of 18 members projected ≥1 hike. Market reaction was risk-off (2-yr yield +16bp; Dow -500 on Jun 16). Next **FOMC decision Jul 29** (just outside the 14-day window). Higher-for-longer is a headwind for the rate-sensitive growth names. [Fed statement](https://www.federalreserve.gov/newsevents/pressreleases/monetary20260617a.htm) · [dot plot](https://finance.yahoo.com/economy/policy/article/fed-dot-plot-almost-half-of-fomc-members-project-at-least-one-interest-rate-hike-this-year-183645064.html)

**Semiconductor cycle (MU, NVDA, AVGO, INTC, SMCI):** Memory is in a tight/"supercycle" setup — SK Hynix and others describe HBM/DRAM/NAND as essentially **sold out for 2026**, and Micron (only US-HQ'd HBM maker) reports **today**. Offsetting this, an **AI-stock sell-off** — triggered partly by AVGO's Jun 3 *no-raise* on its AI target (PHLX -10% on Jun 5) — has pressured NVDA/AMD/MU and was still live into Jun 22–23. China export controls remain a fluid overhang for NVDA (H200 China revenue still zero; approval uncertain) and were extended extraterritorially to Chinese firms on Jun 1. [memory supercycle](https://www.spglobal.com/market-intelligence/en/news-insights/research/2026/01/ai-memory-boom-squeezes-legacy-dram-supply-pushing-prices-higher) · [sell-off trigger](https://www.kavout.com/market-lens/what-triggered-the-recent-semiconductor-sell-off)

**AI capex / data-center sentiment (NVDA, AVGO, AMZN, GOOGL, META spenders; SMCI, VRT, CEG downstream):** Big-5 2026 capex estimated ~$700–725B (~64% above 2025, ~75% to AI infra; AMZN/MSFT/GOOGL/META each >$100B). Narrative is still **supply-constrained, not demand-constrained** (Goldman flags an >11 GW US data-center capacity shortfall today, widening toward ~45 GW by 2028). But there's clear **intra-AI selectivity** — on Jun 22 SMCI surged while GOOGL and PLTR fell the same session. [capex tracker](https://www.goldmansachs.com/insights/articles/tracking-trillions-the-assumptions-shaping-scale-of-the-ai-build-out)

**Power / electricity demand (CEG, VRT; offtakers META/AMZN/GOOGL):** The data-center power theme keeps producing long-dated deals — **CEG signed a Walmart nuclear PPA (Jun 23)** on top of its Meta 20-yr/1.1GW Clinton deal, while **VRT** keeps broadening its power+cooling stack (ThermoKey close Jun 12). PJM capacity prices have surged. Competition is intensifying (Talen–Amazon 1,920 MW, Jun 11). [power deals](https://finance.yahoo.com/sectors/energy/articles/ai-data-center-power-deals-181559704.html)

**Scheduled macro events in the window:**
- **Jun 24 (today, AMC)** — Micron fiscal Q3 earnings (memory read-through)
- **~Jun 25** — Personal Income & Outlays / May PCE (inflation gauge; some calendars say Jun 26 — confirm)
- **~Jul 2 (Thu)** — June jobs report (BLS; likely Thu due to Jul 4 holiday)
- **~Jul 1–3** — Tesla Q2 production/delivery report
- *(Outside window: June CPI ~Jul 15; FOMC decision Jul 29)*

*Caveat: several primary IR/SEC and federalreserve.gov/bls.gov pages returned HTTP 403 to direct fetch; some dates and figures rest on reputable secondary reporting and should be confirmed against primary calendars/releases. Unconfirmed/rumor items are labeled throughout (notably: TSLA–SpaceX merger talk; exact META EU fine amounts; Intel–Apple deal scope; NVDA H200 China revenue; AMZN FTC suit; several earnings dates).*
