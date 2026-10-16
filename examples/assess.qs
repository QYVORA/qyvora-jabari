# assess.qs — a runnable example of the shared QYVORA scripting engine driving
# jabari's offline APK analysis pipeline.
#
#     jabari script run examples/assess.qs
#
# Run from the jabari repository root so the relative fixture path resolves.
# jabari has no simulation mode: the offline path is static APK analysis of the
# bundled fixture, which is the same tiny debuggable package jabari's own unit
# tests build.

# Select the offline fixture APK as the target (authorization is recorded the
# same way --authorized/-y records it).
set assess.apk -> "examples/fixtures/debuggable.apk"
set assess.profile -> "application"

# Run the full APK pipeline (apk-analysis, analysis, validation, risk).
run assess

# Read live properties from the completed run via $(...).
echo -> "session  = $(session.id)"
echo -> "target   = $(target.name)"
echo -> "apps     = $(result.apps)"
echo -> "findings = $(result.findings)"
echo -> "score    = $(result.score) ($(result.level))"

# Show capability metadata and write a Markdown report artifact.
show jabari.apk
report -> "jabari-report"

echo -> "done"
