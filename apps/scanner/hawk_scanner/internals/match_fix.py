def match_strings(args, content, source='text'):
    redacted = False
    if args and 'connection' in args:
        connections = get_connection(args)
        if 'notify' in connections:
            redacted = connections.get('notify', {}).get('redacted', False)

    patterns = get_fingerprint_file(args)

    from sdk.engine import SharedAnalyzerEngine
    wrapper = SharedAnalyzerEngine()
    engine = wrapper.get_engine()

    results = engine.analyze(text=content, entities=None, language="en")

    findings = []
    for r in results:
        findings.append({
            "pattern_name": r.entity_type,
            "confidence_score": r.score,
            "match": content[r.start:r.end],
            "start": r.start,
            "end": r.end
        })

    matched_strings = []

    for finding in findings:
        matched_strings.append(finding)
        if args:
            print_debug(
                args,
                f"Found pattern: {finding['pattern_name']} Score: {finding['confidence_score']}"
            )

    return matched_strings
