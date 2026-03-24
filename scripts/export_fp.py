import os
import psycopg2
import json
from datetime import datetime

# DB Config
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_NAME = os.getenv("DB_NAME", "arc_hawk")
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASS = os.getenv("DB_PASS", "postgres")

def export_false_positives():
    try:
        conn = psycopg2.connect(
            host=DB_HOST, port=DB_PORT, dbname=DB_NAME, user=DB_USER, password=DB_PASS
        )
        cur = conn.cursor()
        
        query = """
        SELECT f.pattern_name, f.sample_text, fb.comments
        FROM finding_feedback fb
        JOIN findings f ON fb.finding_id = f.id
        WHERE fb.feedback_type = 'FALSE_POSITIVE'
        """
        
        cur.execute(query)
        rows = cur.fetchall()
        
        exclusion_patterns = []
        for row in rows:
            exclusion_patterns.append({
                "pattern_name": row[0],
                "sample_text": row[1],
                "reason": row[2],
                "exported_at": datetime.now().isoformat()
            })
            
        with open("false_positive_exclusions.json", "w") as f:
            json.dump(exclusion_patterns, f, indent=2)
            
        print(f"Successfully exported {len(exclusion_patterns)} false positives to false_positive_exclusions.json")
        
    except Exception as e:
        print(f"Error: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    export_false_positives()
