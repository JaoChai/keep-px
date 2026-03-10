import { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import {
  Search,
  BookOpen,
  Radio,
  Activity,
  RotateCcw,
  FileText,
  CreditCard,
  Settings,
  ChevronDown,
  Zap,
  Globe,
  Eye,
  Play,
  AlertTriangle,
  CheckCircle2,
  LogIn,
  ChevronsUpDown,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface GuideSection {
  id: string
  icon: React.ElementType
  iconColor: string
  iconBg: string
  title: string
  badge?: string
  description: string
  subsections: GuideSubsection[]
}

interface GuideSubsection {
  id: string
  title: string
  content: React.ReactNode
}

// ---------------------------------------------------------------------------
// Flow Step Component
// ---------------------------------------------------------------------------

function FlowStep({ step, label, last }: { step: number; label: string; last?: boolean }) {
  return (
    <div className="flex items-start gap-3">
      <div className="flex flex-col items-center">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
          {step}
        </div>
        {!last && <div className="w-px grow bg-border mt-1 min-h-[20px]" />}
      </div>
      <p className="text-sm text-foreground pt-1">{label}</p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Info Box Component
// ---------------------------------------------------------------------------

function InfoBox({ type, children }: { type: 'tip' | 'warning' | 'important'; children: React.ReactNode }) {
  const styles = {
    tip: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-400',
    warning: 'border-amber-500/30 bg-amber-500/5 text-amber-700 dark:text-amber-400',
    important: 'border-blue-500/30 bg-blue-500/5 text-blue-700 dark:text-blue-400',
  }
  const icons = {
    tip: CheckCircle2,
    warning: AlertTriangle,
    important: Zap,
  }
  const labels = { tip: 'Tips', warning: 'Warning', important: 'Note' }
  const Icon = icons[type]
  return (
    <div className={cn('flex gap-3 rounded-lg border p-3 text-sm', styles[type])}>
      <Icon className="h-4 w-4 mt-0.5 shrink-0" />
      <div>
        <span className="font-semibold">{labels[type]}:</span> {children}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Table Component
// ---------------------------------------------------------------------------

function GuideTable({ headers, rows }: { headers: string[]; rows: string[][] }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/50">
            {headers.map((h, i) => (
              <th key={i} className="px-4 py-2 text-left font-medium text-muted-foreground">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-b border-border last:border-b-0">
              {row.map((cell, j) => (
                <td key={j} className="px-4 py-2 text-foreground">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Animated Collapsible Wrapper
// ---------------------------------------------------------------------------

function AnimatedCollapse({ open, children }: { open: boolean; children: React.ReactNode }) {
  const contentRef = useRef<HTMLDivElement>(null)
  const [height, setHeight] = useState<number | undefined>(open ? undefined : 0)

  useEffect(() => {
    if (!contentRef.current) return
    if (open) {
      setHeight(contentRef.current.scrollHeight)
      const timer = setTimeout(() => setHeight(undefined), 200)
      return () => clearTimeout(timer)
    } else {
      const h = contentRef.current.scrollHeight
      requestAnimationFrame(() => {
        setHeight(h)
        requestAnimationFrame(() => setHeight(0))
      })
    }
  }, [open])

  return (
    <div
      className="overflow-hidden transition-[height] duration-200 ease-in-out"
      style={{ height: height === undefined ? 'auto' : height }}
    >
      <div ref={contentRef}>{children}</div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Guide Content Data
// ---------------------------------------------------------------------------

const guideSections: GuideSection[] = [
  {
    id: 'getting-started',
    icon: LogIn,
    iconColor: 'text-blue-600 dark:text-blue-400',
    iconBg: 'bg-blue-100 dark:bg-blue-900/40',
    title: 'Getting Started',
    badge: 'Start Here',
    description: 'Sign up and learn the first steps',
    subsections: [
      {
        id: 'login',
        title: 'Sign Up & Log In',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Go to Keep-PX and click <strong className="text-foreground">"Sign in with Google"</strong> — your account is created automatically. After logging in, you'll be directed to the dashboard immediately.
            </p>
          </div>
        ),
      },
      {
        id: 'first-steps',
        title: 'First Steps After Login',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="Go to Pixels > Create your first Pixel" />
            <FlowStep step={2} label="Go to Sale Pages > Create a sale page & link your Pixel" />
            <FlowStep step={3} label="Publish the page > Share the link with your audience" />
            <FlowStep step={4} label="Check Events and Dashboard for your data" last />
          </div>
        ),
      },
    ],
  },
  {
    id: 'dashboard',
    icon: Eye,
    iconColor: 'text-purple-600 dark:text-purple-400',
    iconBg: 'bg-purple-100 dark:bg-purple-900/40',
    title: 'Dashboard',
    description: 'Overview and analytics at a glance',
    subsections: [
      {
        id: 'dashboard-overview',
        title: 'Summary Cards',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">The dashboard shows your overview through 5 main cards:</p>
            <GuideTable
              headers={['Card', 'Description']}
              rows={[
                ['Active Pixels', 'Active / Total Pixel count'],
                ['Events Today', 'Today\'s events (with trend vs yesterday)'],
                ['CAPI Rate', 'Successful event send rate to Facebook'],
                ['Events This Week', 'Event count this week'],
                ['Active Replays', 'Currently running replays'],
              ]}
            />
            <InfoBox type="tip">
              CAPI Rate is color-coded: green = good, yellow = moderate, red = needs attention
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'dashboard-quota',
        title: 'Monthly Event Quota',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              The progress bar shows events used vs your plan's limit. If approaching the limit, consider upgrading or purchasing Add-ons.
            </p>
          </div>
        ),
      },
      {
        id: 'dashboard-chart',
        title: 'Charts & Data',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              The Event Volume chart shows trends over time — choose from 7d, 14d, 30d, or 90d views. Also see recent activity, Pixel status, and top event types.
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'pixels',
    icon: Radio,
    iconColor: 'text-emerald-600 dark:text-emerald-400',
    iconBg: 'bg-emerald-100 dark:bg-emerald-900/40',
    title: 'Pixel Management',
    description: 'Create, configure, and test your Pixels',
    subsections: [
      {
        id: 'pixel-create',
        title: 'Create a New Pixel',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">Click "Create Pixel" and fill in the details:</p>
            <GuideTable
              headers={['Field', 'Description', 'Example']}
              rows={[
                ['Pixel Name', 'Internal name for easy reference', 'Clothing Ads Pixel'],
                ['Facebook Pixel ID', 'Pixel ID from Events Manager', '123456789012345'],
                ['Access Token', 'Token for CAPI', 'EAAxxxxxxxx...'],
                ['Test Event Code', '(Optional) Test code', 'TEST12345'],
                ['Backup Pixel', '(Optional) Backup pixel', 'Select from list'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'pixel-find-credentials',
        title: 'Finding Pixel ID & Access Token',
        content: (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground mb-3">Find these in Facebook Events Manager:</p>
            <FlowStep step={1} label="Open Facebook Events Manager > Select your Pixel" />
            <FlowStep step={2} label="Pixel ID: Shown below the Pixel name (15-16 digits)" />
            <FlowStep step={3} label="Access Token: Go to Settings > Generate Access Token" last />
          </div>
        ),
      },
      {
        id: 'pixel-actions',
        title: 'Managing Pixels',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Action', 'What It Does']}
              rows={[
                ['Test Connection', 'Test Facebook CAPI connection'],
                ['Edit', 'Modify Pixel settings'],
                ['Delete', 'Remove Pixel (stored events remain)'],
                ['Toggle Status', 'Enable/Disable Pixel temporarily'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'pixel-backup',
        title: 'Backup Pixel',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              With Backup Pixel enabled, events are sent to both the primary and backup Pixels <strong className="text-foreground">simultaneously</strong> via CAPI. If your primary gets banned, the backup still has all the data.
            </p>
            <InfoBox type="important">
              Create Pixel B first, then edit Pixel A to select Pixel B as its backup.
            </InfoBox>
          </div>
        ),
      },
    ],
  },
  {
    id: 'events',
    icon: Activity,
    iconColor: 'text-orange-600 dark:text-orange-400',
    iconBg: 'bg-orange-100 dark:bg-orange-900/40',
    title: 'Event Tracking',
    description: 'Real-time and historical event data',
    subsections: [
      {
        id: 'events-live',
        title: 'Live Mode (Real-time)',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Watch events arrive in real-time as visitors interact with your sale pages.
            </p>
            <div className="flex flex-wrap gap-2">
              <Badge variant="secondary" className="gap-1"><Play className="h-3 w-3" /> Play/Pause</Badge>
              <Badge variant="secondary" className="gap-1">Refresh</Badge>
              <Badge variant="secondary" className="gap-1">Clear</Badge>
            </div>
            <GuideTable
              headers={['Column', 'Example']}
              rows={[
                ['Event Name', 'PageView, Purchase, Lead, ViewContent'],
                ['Pixel', 'Pixel that received the event'],
                ['Source URL', 'Page where the event occurred'],
                ['CAPI', 'Success / Failed'],
                ['Time', '2 minutes ago'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'events-history',
        title: 'History Mode',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Browse all past events with Pixel filter and pagination (50 events/page).
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'replay',
    icon: RotateCcw,
    iconColor: 'text-rose-600 dark:text-rose-400',
    iconBg: 'bg-rose-100 dark:bg-rose-900/40',
    title: 'Replay Center',
    badge: 'Important',
    description: 'Re-send events to a new Pixel after a ban',
    subsections: [
      {
        id: 'replay-what',
        title: 'What is Replay?',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Replay re-sends stored events to a new Pixel via Facebook CAPI.
            </p>
            <p className="text-sm font-medium text-foreground">When to use:</p>
            <ul className="text-sm text-muted-foreground space-y-1 ml-4 list-disc">
              <li>Ad account banned — migrate data to a new Pixel</li>
              <li>Want to re-send events so Facebook learns faster</li>
            </ul>
          </div>
        ),
      },
      {
        id: 'replay-create',
        title: 'Creating a Replay',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Field', 'Description']}
              rows={[
                ['Source Pixel', 'Source Pixel (pull events from here)'],
                ['Target Pixel', 'Destination Pixel (send events here)'],
                ['Event Type', '(Optional) Filter specific event types'],
                ['Date Range', '(Optional) Set time range'],
                ['Time Mode', 'Original = keep timestamps / Current = use current time'],
                ['Batch Delay', 'Delay between batches (0-60,000 ms)'],
              ]}
            />
            <InfoBox type="warning">
              Events older than 7 days: use Time Mode = "Current" since Facebook may reject old timestamps.
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'replay-status',
        title: 'Replay Status',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Status', 'Meaning']}
              rows={[
                ['Pending', 'Waiting to start'],
                ['Running', 'Sending events'],
                ['Completed', 'All events sent successfully'],
                ['Failed', 'Send failed (check error details)'],
                ['Cancelled', 'Manually cancelled'],
              ]}
            />
            <p className="text-sm text-muted-foreground">
              <strong className="text-foreground">Cancel</strong> stops a running replay.{' '}
              <strong className="text-foreground">Retry Failed</strong> re-sends failed events.
            </p>
          </div>
        ),
      },
      {
        id: 'replay-credit',
        title: 'Replay Credits',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Each replay uses <strong className="text-foreground">1 Credit</strong>. Check your balance on the Replay or Billing page. Purchase more at Billing &rarr; Replays tab.
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'sale-pages',
    icon: FileText,
    iconColor: 'text-pink-600 dark:text-pink-400',
    iconBg: 'bg-pink-100 dark:bg-pink-900/40',
    title: 'Sale Pages',
    description: 'Create landing pages with auto-tracking',
    subsections: [
      {
        id: 'salepage-what',
        title: 'What are Sale Pages?',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Sale pages are hosted landing pages created by Keep-PX for showcasing products/services. They <strong className="text-foreground">automatically capture Pixel events</strong> when visited. Share the link on social media or email.
            </p>
          </div>
        ),
      },
      {
        id: 'salepage-templates',
        title: 'Templates',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Template', 'Description']}
              rows={[
                ['Classic', 'Fixed layout — fill in the fields, simple and clean'],
                ['Blocks', 'Drag-and-drop — fully customizable and flexible'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'salepage-settings',
        title: 'Page Settings',
        content: (
          <div className="space-y-4">
            <p className="text-sm font-medium text-foreground">Basic Info</p>
            <GuideTable
              headers={['Field', 'Description']}
              rows={[
                ['Page Name', 'Internal name (visitors don\'t see this)'],
                ['URL Slug', 'URL path e.g. my-product > /p/my-product'],
                ['Link Pixels', 'Select Pixels for tracking (multi-select)'],
              ]}
            />
            <p className="text-sm font-medium text-foreground mt-4">Tracking</p>
            <GuideTable
              headers={['Field', 'Description']}
              rows={[
                ['CTA Event', 'Event fired on button click: Lead / Purchase / Contact / CompleteRegistration'],
                ['Content Name', 'Product name (sent to Facebook)'],
                ['Content Value', 'Product price (sent to Facebook)'],
                ['Currency', 'Currency: THB, USD'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'salepage-auto-events',
        title: 'Auto-fired Events',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">Events are fired automatically when visitors interact:</p>
            <FlowStep step={1} label="Visitor opens page > PageView + ViewContent" />
            <FlowStep step={2} label="Visitor clicks CTA > Purchase / Lead / Contact" />
            <FlowStep step={3} label="Visitor clicks LINE / Phone > Contact" last />
            <InfoBox type="tip">
              All events are sent via Facebook CAPI automatically — no code needed.
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'salepage-publish',
        title: 'Publishing',
        content: (
          <div className="space-y-3">
            <GuideTable
              headers={['Action', 'Description']}
              rows={[
                ['Save as Draft', 'Save without publishing'],
                ['Publish', 'Go live — visitors can access via the link'],
              ]}
            />
          </div>
        ),
      },
    ],
  },
  {
    id: 'billing',
    icon: CreditCard,
    iconColor: 'text-amber-600 dark:text-amber-400',
    iconBg: 'bg-amber-100 dark:bg-amber-900/40',
    title: 'Billing',
    description: 'Plans, credits, and subscriptions',
    subsections: [
      {
        id: 'billing-plans',
        title: 'Plans',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Plan', 'Events/Month', 'Description']}
              rows={[
                ['Sandbox (Free)', 'Limited', 'Trial'],
                ['Launch', '1M', 'For getting started'],
                ['Shield', '5M', 'For medium businesses'],
                ['Vault', 'Unlimited', 'For large businesses'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'billing-replay-packs',
        title: 'Replay Credit Packs',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Purchase additional Replay Credits in the Replays tab on the Billing page. Available packs: 1 credit, 3 credits, and Unlimited. Click "Buy Pack" and pay via Stripe.
            </p>
          </div>
        ),
      },
      {
        id: 'billing-addons',
        title: 'Add-ons',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">Monthly subscription add-ons:</p>
            <GuideTable
              headers={['Add-on', 'What You Get']}
              rows={[
                ['Events +1M', '+1 million event quota/month'],
                ['Sale Pages +10', '+10 sale pages'],
                ['Pixels +10', '+10 pixels'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'billing-manage',
        title: 'Manage Billing',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Click "Manage Billing" to open the Stripe Customer Portal for changing payment method, viewing invoices, or cancelling subscriptions.
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'settings',
    icon: Settings,
    iconColor: 'text-slate-600 dark:text-slate-400',
    iconBg: 'bg-slate-100 dark:bg-slate-800/60',
    title: 'Account Settings',
    description: 'Profile and API key management',
    subsections: [
      {
        id: 'settings-profile',
        title: 'Profile',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Shows your name, email from Google Account, and current plan (read-only).
            </p>
          </div>
        ),
      },
      {
        id: 'settings-api-key',
        title: 'API Key',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              The API Key is used by sale pages to send events to the system.
            </p>
            <GuideTable
              headers={['Action', 'What It Does']}
              rows={[
                ['Show/Hide', 'Toggle API Key visibility'],
                ['Copy', 'Copy API Key to clipboard'],
                ['Regenerate', 'Create new key (old key stops working immediately)'],
              ]}
            />
            <InfoBox type="warning">
              After regenerating, sale pages using the old key won't be able to send events. System-created sale pages are updated automatically.
            </InfoBox>
          </div>
        ),
      },
    ],
  },
  {
    id: 'glossary',
    icon: BookOpen,
    iconColor: 'text-teal-600 dark:text-teal-400',
    iconBg: 'bg-teal-100 dark:bg-teal-900/40',
    title: 'Glossary',
    description: 'Key terms and definitions',
    subsections: [
      {
        id: 'glossary-pixel',
        title: 'Facebook Pixel',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Facebook's tracking code for collecting visitor behavior data. Consists of a <strong className="text-foreground">Pixel ID</strong> (15-16 digit number) and an <strong className="text-foreground">Access Token</strong> (key for sending data to Facebook).
            </p>
          </div>
        ),
      },
      {
        id: 'glossary-events',
        title: 'Events',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">Actions visitors take on your website:</p>
            <GuideTable
              headers={['Event', 'Meaning']}
              rows={[
                ['PageView', 'Opened a page'],
                ['ViewContent', 'Viewed product content'],
                ['Lead', 'Showed interest (filled form, clicked details)'],
                ['Purchase', 'Made a purchase'],
                ['Contact', 'Contacted the store (LINE, phone)'],
                ['CompleteRegistration', 'Completed sign-up'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'glossary-capi',
        title: 'Conversions API (CAPI)',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Server-to-Server method for sending events to Facebook. More reliable than browser-based tracking, not blocked by ad blockers, and more accurate. Keep-PX uses CAPI as the primary event delivery method.
            </p>
          </div>
        ),
      },
      {
        id: 'glossary-other',
        title: 'Other Terms',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Term', 'Definition']}
              rows={[
                ['Backup Pixel', 'Secondary Pixel that receives events alongside the primary via CAPI'],
                ['Replay', 'Re-send stored events to a new Pixel'],
                ['Replay Credit', 'Unit counting how many replays you can run'],
                ['Event Quota', 'Maximum events per month, based on your plan'],
              ]}
            />
          </div>
        ),
      },
    ],
  },
  {
    id: 'scenarios',
    icon: Globe,
    iconColor: 'text-indigo-600 dark:text-indigo-400',
    iconBg: 'bg-indigo-100 dark:bg-indigo-900/40',
    title: 'Real-World Scenarios',
    description: 'Step-by-step guides for common situations',
    subsections: [
      {
        id: 'scenario-new',
        title: 'First-Time Setup',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="Sign in with Google" />
            <FlowStep step={2} label="Go to Pixels > Create Pixel (enter Pixel ID + Access Token)" />
            <FlowStep step={3} label="Click Test Connection to verify Facebook connectivity" />
            <FlowStep step={4} label="Go to Sale Pages > Create page > Link Pixel" />
            <FlowStep step={5} label="Click Publish > Copy link /p/xxx" />
            <FlowStep step={6} label="Share link on Facebook, LINE, email" />
            <FlowStep step={7} label="Check Events > Live Mode to watch events arrive" last />
          </div>
        ),
      },
      {
        id: 'scenario-banned',
        title: 'Account Banned — Data Recovery',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="Create a new Facebook ad account + new Pixel" />
            <FlowStep step={2} label="Go to Pixels > Create the new Pixel in Keep-PX" />
            <FlowStep step={3} label="Go to Replay Center" />
            <FlowStep step={4} label="Set Source = old Pixel, Target = new Pixel" />
            <FlowStep step={5} label="(If events > 7 days old) Select Time Mode = Current" />
            <FlowStep step={6} label="Click Preview > Review event count > Confirm" />
            <FlowStep step={7} label="Wait for Replay to complete > Check status panel > Done!" last />
            <InfoBox type="tip">
              To replay only specific event types (e.g. Purchase), set the Event Type filter before clicking Preview.
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'scenario-backup',
        title: 'Proactive Protection with Backup Pixel',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="Create primary Pixel (Pixel A)" />
            <FlowStep step={2} label="Create backup Pixel (Pixel B) in a different ad account" />
            <FlowStep step={3} label="Edit Pixel A > Select Pixel B as Backup" />
            <FlowStep step={4} label="All events are now sent to both Pixel A and B simultaneously" />
            <FlowStep step={5} label="If Pixel A gets banned > Pixel B still has all the data" last />
          </div>
        ),
      },
    ],
  },
]

// Quick-start cards config
const quickStartCards = [
  { sectionId: 'pixels', icon: Radio, label: 'Create Pixel', color: 'text-emerald-600 dark:text-emerald-400', bg: 'bg-emerald-100 dark:bg-emerald-900/40' },
  { sectionId: 'sale-pages', icon: FileText, label: 'Create Sale Page', color: 'text-pink-600 dark:text-pink-400', bg: 'bg-pink-100 dark:bg-pink-900/40' },
  { sectionId: 'replay', icon: RotateCcw, label: 'Replay Events', color: 'text-rose-600 dark:text-rose-400', bg: 'bg-rose-100 dark:bg-rose-900/40' },
  { sectionId: 'glossary', icon: BookOpen, label: 'Glossary', color: 'text-teal-600 dark:text-teal-400', bg: 'bg-teal-100 dark:bg-teal-900/40' },
]

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function GuidePage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['getting-started']))

  // Filter sections by search query
  const filteredSections = useMemo(() => {
    if (!searchQuery.trim()) return guideSections
    const q = searchQuery.toLowerCase()
    return guideSections
      .map((section) => {
        const sectionMatch = section.title.toLowerCase().includes(q) || section.description.toLowerCase().includes(q)
        const matchedSubs = section.subsections.filter((sub) => sub.title.toLowerCase().includes(q))
        if (sectionMatch) return section
        if (matchedSubs.length > 0) return { ...section, subsections: matchedSubs }
        return null
      })
      .filter(Boolean) as GuideSection[]
  }, [searchQuery])

  const handleSearchChange = useCallback((value: string) => {
    setSearchQuery(value)
    if (value.trim()) {
      const q = value.toLowerCase()
      const matched = guideSections.filter((s) =>
        s.title.toLowerCase().includes(q) ||
        s.description.toLowerCase().includes(q) ||
        s.subsections.some((sub) => sub.title.toLowerCase().includes(q))
      )
      setExpandedSections(new Set(matched.map((s) => s.id)))
    }
  }, [])

  const toggleSection = useCallback((id: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const scrollToSection = useCallback((sectionId: string) => {
    setExpandedSections((prev) => new Set([...prev, sectionId]))
    // Small delay so the section expands before scrolling
    setTimeout(() => {
      const el = document.getElementById(`section-${sectionId}`)
      el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 50)
  }, [])

  const allExpanded = expandedSections.size === guideSections.length
  const toggleAll = useCallback(() => {
    if (allExpanded) {
      setExpandedSections(new Set())
    } else {
      setExpandedSections(new Set(guideSections.map((s) => s.id)))
    }
  }, [allExpanded])

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="max-w-3xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <BookOpen className="h-5 w-5 text-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-foreground">User Guide</h1>
              <p className="text-sm text-muted-foreground">Everything you need to know about Keep-PX</p>
            </div>
          </div>
        </div>

        {/* Quick-Start Cards */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
          {quickStartCards.map((card) => (
            <button
              key={card.sectionId}
              onClick={() => scrollToSection(card.sectionId)}
              aria-label={`Jump to ${card.label} section`}
              className="flex flex-col items-center gap-2 rounded-xl border border-border bg-card p-4 hover:bg-accent/50 hover:border-border/80 transition-colors cursor-pointer group"
            >
              <div className={cn('flex h-10 w-10 items-center justify-center rounded-lg transition-transform group-hover:scale-105', card.bg)}>
                <card.icon className={cn('h-5 w-5', card.color)} />
              </div>
              <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors text-center">
                {card.label}
              </span>
            </button>
          ))}
        </div>

        {/* Search + Expand All */}
        <div className="flex items-center gap-2 mb-6">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search topics... e.g. Pixel, Replay, Sale Page"
              className="pl-9 h-10"
            />
            {searchQuery && (
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
                {filteredSections.length} found
              </span>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={toggleAll}
            className="shrink-0 gap-1.5 h-10 cursor-pointer"
          >
            <ChevronsUpDown className="h-4 w-4" />
            <span className="hidden sm:inline">{allExpanded ? 'Collapse All' : 'Expand All'}</span>
          </Button>
        </div>

        {/* Horizontal Pill Nav (visible md+) */}
        <nav aria-label="Guide sections" className="hidden md:flex items-center gap-1.5 mb-6 overflow-x-auto pb-1 scrollbar-thin">
          {guideSections.map((section) => (
            <button
              key={section.id}
              onClick={() => scrollToSection(section.id)}
              aria-current={expandedSections.has(section.id) ? 'true' : undefined}
              className={cn(
                'flex items-center gap-1.5 shrink-0 rounded-full px-3 py-1.5 text-xs font-medium transition-colors cursor-pointer border',
                expandedSections.has(section.id)
                  ? 'bg-accent text-accent-foreground border-border'
                  : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground border-transparent'
              )}
            >
              <section.icon className="h-3.5 w-3.5" />
              {section.title}
            </button>
          ))}
        </nav>

        {/* No Results */}
        {filteredSections.length === 0 && (
          <div className="text-center py-12">
            <Search className="h-10 w-10 text-muted-foreground/30 mx-auto mb-3" />
            <p className="text-sm text-muted-foreground">No topics found</p>
            <button
              onClick={() => setSearchQuery('')}
              className="text-sm text-primary hover:underline mt-1 cursor-pointer"
            >
              Clear search
            </button>
          </div>
        )}

        {/* Sections — Single-Level Accordion */}
        <div className="space-y-3">
          {filteredSections.map((section) => {
            const isExpanded = expandedSections.has(section.id)
            return (
              <div
                key={section.id}
                id={`section-${section.id}`}
                className="rounded-xl border border-border bg-card overflow-hidden"
              >
                {/* Section Header */}
                <button
                  onClick={() => toggleSection(section.id)}
                  aria-expanded={isExpanded}
                  aria-controls={`section-content-${section.id}`}
                  className="flex items-center justify-between w-full px-5 py-4 hover:bg-accent/30 transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className={cn('flex h-9 w-9 shrink-0 items-center justify-center rounded-lg', section.iconBg)}>
                      <section.icon className={cn('h-5 w-5', section.iconColor)} />
                    </div>
                    <div className="min-w-0 text-left">
                      <div className="flex items-center gap-2">
                        <span className="text-base font-semibold text-foreground">{section.title}</span>
                        {section.badge && (
                          <Badge variant="secondary" className="text-[10px] px-1.5 py-0">{section.badge}</Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground truncate">{section.description}</p>
                    </div>
                  </div>
                  <ChevronDown
                    className={cn(
                      'h-4 w-4 text-muted-foreground transition-transform duration-200 shrink-0 ml-2',
                      isExpanded && 'rotate-180'
                    )}
                  />
                </button>

                {/* Subsections Content — Flat (no nested accordion) */}
                <AnimatedCollapse open={isExpanded}>
                  <div id={`section-content-${section.id}`} role="region" className="border-t border-border px-5 py-5 space-y-6">
                    {section.subsections.map((sub) => (
                      <div key={sub.id}>
                        <h3 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-2">
                          <div className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />
                          {sub.title}
                        </h3>
                        {sub.content}
                      </div>
                    ))}
                  </div>
                </AnimatedCollapse>
              </div>
            )
          })}
        </div>

        {/* Footer */}
        <div className="mt-8 mb-4 text-center">
          <p className="text-xs text-muted-foreground">
            Need more help? Contact our support team anytime.
          </p>
        </div>
      </div>
    </div>
  )
}
